// Package service はビジネスロジックを提供します
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// PatrolService は複数プロジェクト自動巡回のインターフェースを定義します
type PatrolService interface {
	// RegisterProject は巡回対象プロジェクトを登録します
	RegisterProject(path string) error
	// UnregisterProject は巡回対象プロジェクトを解除します
	UnregisterProject(path string) error
	// ListProjects は登録済みプロジェクト一覧を返します
	ListProjects() []PatrolProject
	// ScanProjects は全登録プロジェクトをスキャンし結果を返します
	ScanProjects() []ScanResult
	// StartPatrol は巡回を開始します
	StartPatrol() error
	// StopPatrol は巡回を停止します
	StopPatrol()
	// ResumeProject は承認待ちプロジェクトを再開します
	ResumeProject(projectPath, answer string) error
	// GetStates は全プロジェクトの実行状態を返します
	GetStates() map[string]*ProjectState
	// StartPolling は定期ポーリングを開始します
	StartPolling()
	// StopPolling は定期ポーリングを停止します
	StopPolling()
	// Subscribe はSSEイベントのサブスクリプションを返します
	Subscribe() (<-chan PatrolEvent, func())
}

// patrolServiceImpl はPatrolServiceの実装です
type patrolServiceImpl struct {
	mu            sync.RWMutex
	projects      map[string]PatrolProject // key: path
	states        map[string]*ProjectState // key: path
	slots         chan struct{}            // セマフォ（並列実行数制御）
	claudeService ClaudeService
	ntfyService   NtfyService
	configPath    string             // JSONファイルパス
	patrolRunning bool               // 巡回実行中フラグ
	patrolCancel  context.CancelFunc // 巡回キャンセル用

	subMu       sync.Mutex
	subscribers map[int]chan PatrolEvent
	nextSubID   int

	pollingCancel context.CancelFunc
}

// NewPatrolService は新しいPatrolServiceを生成します
func NewPatrolService(claudeService ClaudeService, ntfyService NtfyService, configPath string) PatrolService {
	s := &patrolServiceImpl{
		projects:      make(map[string]PatrolProject),
		states:        make(map[string]*ProjectState),
		slots:         make(chan struct{}, MaxParallelSlots),
		claudeService: claudeService,
		ntfyService:   ntfyService,
		configPath:    configPath,
		subscribers:   make(map[int]chan PatrolEvent),
	}

	// 設定ファイルからプロジェクト一覧を読み込み
	if err := s.loadConfig(); err != nil {
		log.Printf("[PatrolService] Failed to load config: %v", err)
	}

	return s
}

// RegisterProject は巡回対象プロジェクトを登録します
func (s *patrolServiceImpl) RegisterProject(path string) error {
	log.Printf("[PatrolService] RegisterProject started: path=%s", path)

	// パスの正規化
	cleanPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("absolute path required: %s", path)
	}

	// ディレクトリの存在確認
	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", cleanPath)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 重複チェック
	if _, exists := s.projects[cleanPath]; exists {
		return fmt.Errorf("project already registered: %s", cleanPath)
	}

	project := PatrolProject{
		Path: cleanPath,
		Name: filepath.Base(cleanPath),
	}
	s.projects[cleanPath] = project

	// 設定ファイルに保存
	if err := s.saveConfigLocked(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	log.Printf("[PatrolService] RegisterProject completed: path=%s, name=%s", cleanPath, project.Name)
	return nil
}

// UnregisterProject は巡回対象プロジェクトを解除します
func (s *patrolServiceImpl) UnregisterProject(path string) error {
	log.Printf("[PatrolService] UnregisterProject started: path=%s", path)

	cleanPath := filepath.Clean(path)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.projects[cleanPath]; !exists {
		return fmt.Errorf("project not registered: %s", cleanPath)
	}

	delete(s.projects, cleanPath)
	delete(s.states, cleanPath)

	// 設定ファイルに保存
	if err := s.saveConfigLocked(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	log.Printf("[PatrolService] UnregisterProject completed: path=%s", cleanPath)
	return nil
}

// ListProjects は登録済みプロジェクト一覧を返します
func (s *patrolServiceImpl) ListProjects() []PatrolProject {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projects := make([]PatrolProject, 0, len(s.projects))
	for _, p := range s.projects {
		projects = append(projects, p)
	}

	// パスでソート（安定した順序を保証）
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Path < projects[j].Path
	})

	return projects
}

// ScanProjects は全登録プロジェクトをスキャンし結果を返します
func (s *patrolServiceImpl) ScanProjects() []ScanResult {
	log.Printf("[PatrolService] ScanProjects started")

	projects := s.ListProjects()
	results := make([]ScanResult, 0, len(projects))

	for _, project := range projects {
		result := ScanResult{Project: project}

		// git log --oneline -5 を実行
		gitLog, err := s.getGitLog(project.Path)
		if err != nil {
			log.Printf("[PatrolService] Failed to get git log: path=%s, error=%v", project.Path, err)
		} else {
			result.GitLog = gitLog
		}

		// 未処理タスクを取得
		tasks, err := s.getPendingTasks(project.Path)
		if err != nil {
			log.Printf("[PatrolService] Failed to get pending tasks: path=%s, error=%v", project.Path, err)
		} else {
			result.PendingTasks = tasks
		}

		results = append(results, result)
	}

	log.Printf("[PatrolService] ScanProjects completed: projects=%d", len(results))
	return results
}

// StartPatrol は巡回を開始します
func (s *patrolServiceImpl) StartPatrol() error {
	log.Printf("[PatrolService] StartPatrol started")

	s.mu.Lock()
	if s.patrolRunning {
		s.mu.Unlock()
		return fmt.Errorf("patrol is already running")
	}
	s.patrolRunning = true
	ctx, cancel := context.WithCancel(context.Background())
	s.patrolCancel = cancel
	s.mu.Unlock()

	// スキャン実行
	scanResults := s.ScanProjects()

	// スキャン完了イベントを配信
	s.broadcast(PatrolEvent{
		Type:    PatrolEventScanCompleted,
		Message: fmt.Sprintf("Scan completed: %d projects", len(scanResults)),
	})

	// 未処理タスクのあるプロジェクトを並列実行
	var wg sync.WaitGroup
	for _, result := range scanResults {
		if len(result.PendingTasks) == 0 {
			continue
		}

		// 同一プロジェクトが実行中またはWaitingApprovalならスキップ
		s.mu.RLock()
		state, exists := s.states[result.Project.Path]
		var status PatrolStatus
		if exists {
			status = state.Status
		}
		s.mu.RUnlock()
		if exists && (status == StatusRunning || status == StatusWaitingApproval) {
			log.Printf("[PatrolService] Skipping project (already active): path=%s, status=%s", result.Project.Path, status)
			continue
		}

		wg.Add(1)
		go func(sr ScanResult) {
			defer wg.Done()
			// セマフォでスロット取得（contextキャンセルを監視）
			select {
			case s.slots <- struct{}{}:
			case <-ctx.Done():
				log.Printf("[PatrolService] Patrol cancelled, skipping project: path=%s", sr.Project.Path)
				return
			}
			defer func() { <-s.slots }()

			s.startProjectExecution(sr.Project, sr.PendingTasks[0], sr.GitLog)
		}(result)
	}

	// 全プロジェクト完了を待つgoroutine
	go func() {
		wg.Wait()
		s.mu.Lock()
		s.patrolRunning = false
		s.patrolCancel = nil
		s.mu.Unlock()
		cancel() // contextのリソースを解放
		log.Printf("[PatrolService] StartPatrol all projects completed")
	}()

	log.Printf("[PatrolService] StartPatrol dispatched")
	return nil
}

// StopPatrol は巡回を停止します
func (s *patrolServiceImpl) StopPatrol() {
	log.Printf("[PatrolService] StopPatrol called")
	s.mu.Lock()
	s.patrolRunning = false
	if s.patrolCancel != nil {
		s.patrolCancel()
		s.patrolCancel = nil
	}
	s.mu.Unlock()
}

// ResumeProject は承認待ちプロジェクトを再開します
func (s *patrolServiceImpl) ResumeProject(projectPath, answer string) error {
	cleanPath := filepath.Clean(projectPath)
	log.Printf("[PatrolService] ResumeProject started: path=%s", cleanPath)

	s.mu.RLock()
	state, exists := s.states[cleanPath]
	if !exists {
		s.mu.RUnlock()
		return fmt.Errorf("project state not found: %s", cleanPath)
	}
	status := state.Status
	sessionID := state.SessionID
	s.mu.RUnlock()

	if status != StatusWaitingApproval {
		return fmt.Errorf("project is not waiting for approval: %s (status=%s)", cleanPath, status)
	}
	if sessionID == "" {
		return fmt.Errorf("no session ID for project: %s", cleanPath)
	}

	// 状態を実行中に更新
	s.updateState(cleanPath, func(st *ProjectState) {
		st.Status = StatusRunning
		st.Question = nil
	})

	// セマフォでスロット取得して再開
	go func() {
		s.slots <- struct{}{}
		defer func() { <-s.slots }()

		s.resumeProjectExecution(cleanPath, sessionID, answer)
	}()

	return nil
}

// GetStates は全プロジェクトの実行状態を返します
func (s *patrolServiceImpl) GetStates() map[string]*ProjectState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*ProjectState, len(s.states))
	for k, v := range s.states {
		copied := *v
		result[k] = &copied
	}
	return result
}

// StartPolling は定期ポーリングを開始します
func (s *patrolServiceImpl) StartPolling() {
	log.Printf("[PatrolService] StartPolling started")

	s.mu.Lock()
	// 既存のポーリングを停止
	if s.pollingCancel != nil {
		s.pollingCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.pollingCancel = cancel
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(PollingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("[PatrolService] Polling stopped")
				return
			case <-ticker.C:
				log.Printf("[PatrolService] Polling tick: starting patrol")
				if err := s.StartPatrol(); err != nil {
					log.Printf("[PatrolService] Polling patrol failed: %v", err)
				}
			}
		}
	}()
}

// StopPolling は定期ポーリングを停止します
func (s *patrolServiceImpl) StopPolling() {
	log.Printf("[PatrolService] StopPolling called")

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pollingCancel != nil {
		s.pollingCancel()
		s.pollingCancel = nil
	}
}

// Subscribe はSSEイベントのサブスクリプションを返します
func (s *patrolServiceImpl) Subscribe() (<-chan PatrolEvent, func()) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	ch := make(chan PatrolEvent, 100)
	id := s.nextSubID
	s.nextSubID++
	s.subscribers[id] = ch

	unsubscribe := func() {
		s.subMu.Lock()
		defer s.subMu.Unlock()
		delete(s.subscribers, id)
		close(ch)
	}

	return ch, unsubscribe
}

// startProjectExecution はプロジェクトのClaude CLI実行を開始します
func (s *patrolServiceImpl) startProjectExecution(project PatrolProject, taskFile, gitLog string) {
	log.Printf("[PatrolService] startProjectExecution started: path=%s, task=%s", project.Path, taskFile)

	now := time.Now()
	s.mu.Lock()
	s.states[project.Path] = &ProjectState{
		Project:   project,
		Status:    StatusRunning,
		GitLog:    gitLog,
		StartedAt: &now,
		UpdatedAt: &now,
	}
	s.mu.Unlock()

	// 開始イベントを配信
	s.broadcastState(PatrolEventProjectStarted, project.Path)

	// claude -p "/coding @開発/実装/実装待ち/<taskFile>" を実行
	eventCh := make(chan StreamEvent, 100)

	go func() {
		err := s.claudeService.ExecuteCommandStream(context.Background(), project.Path, "coding", "@開発/実装/実装待ち/"+taskFile, nil, eventCh)
		if err != nil {
			log.Printf("[PatrolService] ExecuteCommandStream failed: path=%s, error=%v", project.Path, err)
		}
	}()

	s.monitorStreamEvents(project.Path, eventCh)
}

// resumeProjectExecution は承認待ちプロジェクトのClaude CLI実行を再開します
func (s *patrolServiceImpl) resumeProjectExecution(projectPath, sessionID, answer string) {
	log.Printf("[PatrolService] resumeProjectExecution started: path=%s, sessionID=%s", projectPath, sessionID)

	// 開始イベントを配信
	s.broadcastState(PatrolEventProjectStarted, projectPath)

	eventCh := make(chan StreamEvent, 100)

	go func() {
		err := s.claudeService.ContinueSessionStream(context.Background(), projectPath, sessionID, answer, eventCh)
		if err != nil {
			log.Printf("[PatrolService] ContinueSessionStream failed: path=%s, error=%v", projectPath, err)
		}
	}()

	s.monitorStreamEvents(projectPath, eventCh)
}

// monitorStreamEvents はStreamEventを監視し、状態遷移を管理します
func (s *patrolServiceImpl) monitorStreamEvents(projectPath string, eventCh <-chan StreamEvent) {
	for event := range eventCh {
		switch event.Type {
		case EventTypeQuestion:
			// 承認待ち状態に変更
			var question *Question
			if event.Result != nil && len(event.Result.Questions) > 0 {
				question = &event.Result.Questions[0]
			}

			s.updateState(projectPath, func(st *ProjectState) {
				st.Status = StatusWaitingApproval
				st.SessionID = event.SessionID
				st.Question = question
			})

			// ntfy通知
			if s.ntfyService != nil {
				projectName := filepath.Base(projectPath)
				msg := fmt.Sprintf("Project %s is waiting for approval", projectName)
				if question != nil {
					msg = fmt.Sprintf("[%s] %s", projectName, question.Question)
				}
				s.ntfyService.Notify("Patrol - Approval Required", msg)
			}

			// SSEイベント配信
			s.broadcastState(PatrolEventProjectQuestion, projectPath)

			log.Printf("[PatrolService] Project waiting for approval: path=%s, sessionID=%s", projectPath, event.SessionID)
			return // プロセスは停止されるのでループ終了

		case EventTypeComplete:
			// セッションIDを保持
			if event.SessionID != "" {
				s.updateState(projectPath, func(st *ProjectState) {
					st.SessionID = event.SessionID
				})
			}

		case EventTypeInit:
			if event.SessionID != "" {
				s.updateState(projectPath, func(st *ProjectState) {
					st.SessionID = event.SessionID
				})
			}

		case EventTypeError:
			s.updateState(projectPath, func(st *ProjectState) {
				st.Status = StatusError
				st.Error = event.Message
			})
			s.broadcastState(PatrolEventProjectError, projectPath)
			log.Printf("[PatrolService] Project error: path=%s, error=%s", projectPath, event.Message)
			return
		}
	}

	// チャネルが閉じられた = 完了
	s.updateState(projectPath, func(st *ProjectState) {
		st.Status = StatusCompleted
	})
	s.broadcastState(PatrolEventProjectCompleted, projectPath)
	log.Printf("[PatrolService] Project completed: path=%s", projectPath)
}

// updateState はプロジェクト状態を更新します
func (s *patrolServiceImpl) updateState(projectPath string, fn func(st *ProjectState)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.states[projectPath]
	if !exists {
		state = &ProjectState{
			Project: PatrolProject{
				Path: projectPath,
				Name: filepath.Base(projectPath),
			},
		}
		s.states[projectPath] = state
	}

	fn(state)
	now := time.Now()
	state.UpdatedAt = &now
}

// broadcast はPatrolEventを全subscriberに配信します
func (s *patrolServiceImpl) broadcast(event PatrolEvent) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	for _, ch := range s.subscribers {
		select {
		case ch <- event:
		default:
			// バッファがいっぱいの場合はスキップ
			log.Printf("[PatrolService] Subscriber buffer full, skipping event: type=%s", event.Type)
		}
	}
}

// broadcastState はプロジェクト状態変更イベントを全subscriberに配信します
func (s *patrolServiceImpl) broadcastState(eventType, projectPath string) {
	s.mu.RLock()
	state, exists := s.states[projectPath]
	var stateCopy *ProjectState
	if exists {
		copied := *state
		stateCopy = &copied
	}
	s.mu.RUnlock()

	s.broadcast(PatrolEvent{
		Type:        eventType,
		ProjectPath: projectPath,
		State:       stateCopy,
	})
}

// getGitLog はプロジェクトのgit logを取得します
func (s *patrolServiceImpl) getGitLog(projectPath string) (string, error) {
	cmd := exec.Command("git", "log", "--oneline", "-5")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git log: %w", err)
	}

	return string(output), nil
}

// getPendingTasks は未処理タスクのファイル名一覧を取得します
func (s *patrolServiceImpl) getPendingTasks(projectPath string) ([]string, error) {
	taskDir := filepath.Join(projectPath, "開発", "実装", "実装待ち")

	entries, err := os.ReadDir(taskDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read task directory: %w", err)
	}

	var tasks []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// 隠しファイルはスキップ
		if len(entry.Name()) > 0 && entry.Name()[0] == '.' {
			continue
		}
		tasks = append(tasks, entry.Name())
	}

	return tasks, nil
}

// loadConfig は設定ファイルからプロジェクト一覧を読み込みます
func (s *patrolServiceImpl) loadConfig() error {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // ファイルが存在しない場合は空で開始
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config PatrolConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	for _, p := range config.Projects {
		s.projects[p.Path] = p
	}

	log.Printf("[PatrolService] Loaded %d projects from config: %s", len(config.Projects), s.configPath)
	return nil
}

// saveConfigLocked は設定ファイルにプロジェクト一覧を保存します（mu.Lockを保持した状態で呼ぶこと）
func (s *patrolServiceImpl) saveConfigLocked() error {
	projects := make([]PatrolProject, 0, len(s.projects))
	for _, p := range s.projects {
		projects = append(projects, p)
	}

	// パスでソート（安定した出力を保証）
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Path < projects[j].Path
	})

	config := PatrolConfig{Projects: projects}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// ディレクトリが存在しない場合は作成
	dir := filepath.Dir(s.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// write-to-temp + rename パターンで安全に書き込み
	tmpFile := s.configPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}
	if err := os.Rename(tmpFile, s.configPath); err != nil {
		return fmt.Errorf("failed to rename config: %w", err)
	}

	log.Printf("[PatrolService] Saved %d projects to config: %s", len(projects), s.configPath)
	return nil
}
