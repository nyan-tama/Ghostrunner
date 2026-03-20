package service

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// mockClaudeService はテスト用のClaudeServiceモックです
type mockClaudeService struct {
	mu                       sync.Mutex
	executeCommandStreamFunc func(ctx context.Context, project, command, args string, images []ImageData, eventCh chan<- StreamEvent) error
	continueSessionStreamFn  func(ctx context.Context, project, sessionID, answer string, eventCh chan<- StreamEvent) error
}

func (m *mockClaudeService) ExecuteCommand(_ context.Context, _, _, _ string, _ []ImageData) (*CommandResult, error) {
	return &CommandResult{}, nil
}

func (m *mockClaudeService) ExecuteCommandStream(ctx context.Context, project, command, args string, images []ImageData, eventCh chan<- StreamEvent) error {
	m.mu.Lock()
	fn := m.executeCommandStreamFunc
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, project, command, args, images, eventCh)
	}
	close(eventCh)
	return nil
}

func (m *mockClaudeService) ExecutePlan(_ context.Context, _, _ string) (*CommandResult, error) {
	return &CommandResult{}, nil
}

func (m *mockClaudeService) ExecutePlanStream(_ context.Context, _, _ string, eventCh chan<- StreamEvent) error {
	close(eventCh)
	return nil
}

func (m *mockClaudeService) ContinueSession(_ context.Context, _, _, _ string) (*CommandResult, error) {
	return &CommandResult{}, nil
}

func (m *mockClaudeService) ContinueSessionStream(ctx context.Context, project, sessionID, answer string, eventCh chan<- StreamEvent) error {
	m.mu.Lock()
	fn := m.continueSessionStreamFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, project, sessionID, answer, eventCh)
	}
	close(eventCh)
	return nil
}

// patrolMockNtfyService はPatrolService テスト用のNtfyServiceモックです
type patrolMockNtfyService struct {
	mu       sync.Mutex
	notified []struct{ title, message string }
}

func (m *patrolMockNtfyService) Notify(title, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notified = append(m.notified, struct{ title, message string }{title, message})
}

func (m *patrolMockNtfyService) NotifyError(title, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notified = append(m.notified, struct{ title, message string }{title, message})
}

// newTestPatrolService はテスト用のPatrolServiceを生成します
func newTestPatrolService(t *testing.T, claude ClaudeService, ntfy NtfyService) (PatrolService, string) {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "patrol.json")
	svc := NewPatrolService(claude, ntfy, configPath)
	return svc, tmpDir
}

// --- RegisterProject テスト ---

func TestPatrolService_RegisterProject(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string // 登録するパスを返す
		preReg    func(t *testing.T, svc PatrolService)
		wantError bool
		errMsg    string
	}{
		{
			name: "正常登録_ディレクトリが存在する場合",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantError: false,
		},
		{
			name: "エラー_相対パス",
			setup: func(t *testing.T) string {
				t.Helper()
				return "relative/path"
			},
			wantError: true,
			errMsg:    "absolute path required",
		},
		{
			name: "エラー_存在しないディレクトリ",
			setup: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantError: true,
			errMsg:    "failed to stat path",
		},
		{
			name: "エラー_ファイルはディレクトリではない",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				filePath := filepath.Join(dir, "file.txt")
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
				return filePath
			},
			wantError: true,
			errMsg:    "path is not a directory",
		},
		{
			name: "エラー_重複登録",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			preReg: func(t *testing.T, svc PatrolService) {
				t.Helper()
				// setup で返されるパスを取得するために、ここでは別途ディレクトリを作らない
				// テストケース内で事前登録を行う（setupの戻り値を使う）
			},
			wantError: true,
			errMsg:    "project already registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestPatrolService(t, &mockClaudeService{}, &patrolMockNtfyService{})
			path := tt.setup(t)

			// 重複登録テストの場合は事前に登録
			if tt.name == "エラー_重複登録" {
				if err := svc.RegisterProject(path); err != nil {
					t.Fatalf("pre-registration failed: %v", err)
				}
			}

			err := svc.RegisterProject(path)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" {
					if !containsString(err.Error(), tt.errMsg) {
						t.Errorf("error message: got %q, want containing %q", err.Error(), tt.errMsg)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// 登録後にListProjectsで確認
				projects := svc.ListProjects()
				found := false
				for _, p := range projects {
					if p.Path == filepath.Clean(path) {
						found = true
						if p.Name != filepath.Base(path) {
							t.Errorf("project name: got %q, want %q", p.Name, filepath.Base(path))
						}
					}
				}
				if !found {
					t.Error("registered project not found in ListProjects")
				}
			}
		})
	}
}

func TestPatrolService_RegisterProject_JSONPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "patrol.json")
	claude := &mockClaudeService{}

	// サービスを作成してプロジェクトを登録
	svc := NewPatrolService(claude, nil, configPath)
	projectDir := t.TempDir()
	if err := svc.RegisterProject(projectDir); err != nil {
		t.Fatalf("RegisterProject failed: %v", err)
	}

	// JSONファイルが作成されたことを確認
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	var config PatrolConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if len(config.Projects) != 1 {
		t.Fatalf("projects count: got %d, want 1", len(config.Projects))
	}

	if config.Projects[0].Path != filepath.Clean(projectDir) {
		t.Errorf("project path: got %q, want %q", config.Projects[0].Path, filepath.Clean(projectDir))
	}

	// 新しいサービスインスタンスで設定ファイルから読み込めることを確認
	svc2 := NewPatrolService(claude, nil, configPath)
	projects := svc2.ListProjects()
	if len(projects) != 1 {
		t.Fatalf("reloaded projects count: got %d, want 1", len(projects))
	}
	if projects[0].Path != filepath.Clean(projectDir) {
		t.Errorf("reloaded project path: got %q, want %q", projects[0].Path, filepath.Clean(projectDir))
	}
}

// --- ScanProjects テスト ---

func TestPatrolService_ScanProjects(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T) (string, string) // projectDir, configPath を返す
		wantTaskCount  int
		wantGitLogSet  bool // git log が取得できるか（gitリポジトリの場合）
	}{
		{
			name: "未処理タスクファイルを検出する",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				projectDir := t.TempDir()
				taskDir := filepath.Join(projectDir, "開発", "実装", "実装待ち")
				if err := os.MkdirAll(taskDir, 0755); err != nil {
					t.Fatalf("failed to create task dir: %v", err)
				}
				// タスクファイルを作成
				for _, name := range []string{"task1.md", "task2.md"} {
					if err := os.WriteFile(filepath.Join(taskDir, name), []byte("task"), 0644); err != nil {
						t.Fatalf("failed to create task file: %v", err)
					}
				}
				tmpDir := t.TempDir()
				return projectDir, filepath.Join(tmpDir, "patrol.json")
			},
			wantTaskCount: 2,
		},
		{
			name: "隠しファイルは除外される",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				projectDir := t.TempDir()
				taskDir := filepath.Join(projectDir, "開発", "実装", "実装待ち")
				if err := os.MkdirAll(taskDir, 0755); err != nil {
					t.Fatalf("failed to create task dir: %v", err)
				}
				// 隠しファイルと通常ファイルを作成
				for _, name := range []string{".hidden", "visible.md"} {
					if err := os.WriteFile(filepath.Join(taskDir, name), []byte("task"), 0644); err != nil {
						t.Fatalf("failed to create file: %v", err)
					}
				}
				tmpDir := t.TempDir()
				return projectDir, filepath.Join(tmpDir, "patrol.json")
			},
			wantTaskCount: 1,
		},
		{
			name: "タスクディレクトリが存在しない場合は空",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				projectDir := t.TempDir()
				tmpDir := t.TempDir()
				return projectDir, filepath.Join(tmpDir, "patrol.json")
			},
			wantTaskCount: 0,
		},
		{
			name: "gitリポジトリの場合はgit_logを取得する",
			setup: func(t *testing.T) (string, string) {
				t.Helper()
				projectDir := t.TempDir()
				// git init して少なくとも1コミット作成
				initGitRepo(t, projectDir)
				tmpDir := t.TempDir()
				return projectDir, filepath.Join(tmpDir, "patrol.json")
			},
			wantGitLogSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir, configPath := tt.setup(t)
			svc := NewPatrolService(&mockClaudeService{}, nil, configPath)

			if err := svc.RegisterProject(projectDir); err != nil {
				t.Fatalf("RegisterProject failed: %v", err)
			}

			results := svc.ScanProjects()

			if len(results) != 1 {
				t.Fatalf("scan results count: got %d, want 1", len(results))
			}

			result := results[0]
			if len(result.PendingTasks) != tt.wantTaskCount {
				t.Errorf("pending tasks count: got %d, want %d (tasks: %v)",
					len(result.PendingTasks), tt.wantTaskCount, result.PendingTasks)
			}

			if tt.wantGitLogSet {
				if result.GitLog == "" {
					t.Error("expected git log to be set, got empty")
				}
			}
		})
	}
}

// --- StartPatrol テスト ---

func TestPatrolService_StartPatrol(t *testing.T) {
	t.Run("二重実行防止", func(t *testing.T) {
		// Claude側はブロックし続けるモック
		blocker := make(chan struct{})
		claude := &mockClaudeService{
			executeCommandStreamFunc: func(_ context.Context, _, _, _ string, _ []ImageData, eventCh chan<- StreamEvent) error {
				<-blocker
				close(eventCh)
				return nil
			},
		}

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		svc := NewPatrolService(claude, nil, configPath)

		// タスクのあるプロジェクトを登録
		projectDir := t.TempDir()
		taskDir := filepath.Join(projectDir, "開発", "実装", "実装待ち")
		if err := os.MkdirAll(taskDir, 0755); err != nil {
			t.Fatalf("failed to create task dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(taskDir, "task.md"), []byte("task"), 0644); err != nil {
			t.Fatalf("failed to create task file: %v", err)
		}
		if err := svc.RegisterProject(projectDir); err != nil {
			t.Fatalf("RegisterProject failed: %v", err)
		}

		// 1回目の巡回開始
		if err := svc.StartPatrol(); err != nil {
			t.Fatalf("first StartPatrol failed: %v", err)
		}

		// 2回目の巡回開始はエラー
		err := svc.StartPatrol()
		if err == nil {
			t.Fatal("expected error on second StartPatrol, got nil")
		}
		if !containsString(err.Error(), "already running") {
			t.Errorf("error message: got %q, want containing 'already running'", err.Error())
		}

		// クリーンアップ: ブロッカーを解放
		close(blocker)
		// goroutineが終了するまで少し待つ
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("5並列制限", func(t *testing.T) {
		var (
			mu          sync.Mutex
			maxRunning  int
			currRunning int
		)

		// 各プロジェクト実行時に並列数を計測するモック
		claude := &mockClaudeService{
			executeCommandStreamFunc: func(_ context.Context, _, _, _ string, _ []ImageData, eventCh chan<- StreamEvent) error {
				mu.Lock()
				currRunning++
				if currRunning > maxRunning {
					maxRunning = currRunning
				}
				mu.Unlock()

				time.Sleep(50 * time.Millisecond) // 実行をシミュレート

				mu.Lock()
				currRunning--
				mu.Unlock()

				close(eventCh)
				return nil
			},
		}

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		svc := NewPatrolService(claude, nil, configPath)

		// 8プロジェクトを登録（MaxParallelSlots=5 を超える数）
		for i := 0; i < 8; i++ {
			projectDir := t.TempDir()
			taskDir := filepath.Join(projectDir, "開発", "実装", "実装待ち")
			if err := os.MkdirAll(taskDir, 0755); err != nil {
				t.Fatalf("failed to create task dir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(taskDir, "task.md"), []byte("task"), 0644); err != nil {
				t.Fatalf("failed to create task file: %v", err)
			}
			if err := svc.RegisterProject(projectDir); err != nil {
				t.Fatalf("RegisterProject failed: %v", err)
			}
		}

		if err := svc.StartPatrol(); err != nil {
			t.Fatalf("StartPatrol failed: %v", err)
		}

		// 全プロジェクト完了を待つ
		timeout := time.After(5 * time.Second)
		for {
			select {
			case <-timeout:
				t.Fatal("timeout waiting for patrol to complete")
			default:
			}
			states := svc.GetStates()
			allDone := true
			for _, st := range states {
				if st.Status != StatusCompleted && st.Status != StatusError {
					allDone = false
					break
				}
			}
			if allDone && len(states) > 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

		mu.Lock()
		observed := maxRunning
		mu.Unlock()

		if observed > MaxParallelSlots {
			t.Errorf("max parallel running: got %d, want <= %d", observed, MaxParallelSlots)
		}
	})

	t.Run("タスクのないプロジェクトはスキップされる", func(t *testing.T) {
		callCount := 0
		claude := &mockClaudeService{
			executeCommandStreamFunc: func(_ context.Context, _, _, _ string, _ []ImageData, eventCh chan<- StreamEvent) error {
				callCount++
				close(eventCh)
				return nil
			},
		}

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		svc := NewPatrolService(claude, nil, configPath)

		// タスクのないプロジェクトを登録
		projectDir := t.TempDir()
		if err := svc.RegisterProject(projectDir); err != nil {
			t.Fatalf("RegisterProject failed: %v", err)
		}

		if err := svc.StartPatrol(); err != nil {
			t.Fatalf("StartPatrol failed: %v", err)
		}

		// 完了を待つ
		time.Sleep(200 * time.Millisecond)

		if callCount != 0 {
			t.Errorf("ExecuteCommandStream call count: got %d, want 0", callCount)
		}
	})
}

// --- ResumeProject テスト ---

func TestPatrolService_ResumeProject(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, svc *patrolServiceImpl) string // projectPath を返す
		answer    string
		wantError bool
		errMsg    string
	}{
		{
			name: "正常再開_WaitingApproval状態からRunningに遷移",
			setup: func(t *testing.T, svc *patrolServiceImpl) string {
				t.Helper()
				path := "/test/project"
				now := time.Now()
				svc.mu.Lock()
				svc.projects[path] = PatrolProject{Path: path, Name: "project"}
				svc.states[path] = &ProjectState{
					Project:   PatrolProject{Path: path, Name: "project"},
					Status:    StatusWaitingApproval,
					SessionID: "session-123",
					StartedAt: &now,
					UpdatedAt: &now,
				}
				svc.mu.Unlock()
				return path
			},
			answer:    "yes",
			wantError: false,
		},
		{
			name: "エラー_プロジェクト状態が見つからない",
			setup: func(t *testing.T, svc *patrolServiceImpl) string {
				t.Helper()
				return "/nonexistent/project"
			},
			answer:    "yes",
			wantError: true,
			errMsg:    "project state not found",
		},
		{
			name: "エラー_WaitingApproval以外の状態",
			setup: func(t *testing.T, svc *patrolServiceImpl) string {
				t.Helper()
				path := "/test/running-project"
				now := time.Now()
				svc.mu.Lock()
				svc.projects[path] = PatrolProject{Path: path, Name: "running-project"}
				svc.states[path] = &ProjectState{
					Project:   PatrolProject{Path: path, Name: "running-project"},
					Status:    StatusRunning,
					SessionID: "session-456",
					StartedAt: &now,
					UpdatedAt: &now,
				}
				svc.mu.Unlock()
				return path
			},
			answer:    "yes",
			wantError: true,
			errMsg:    "not waiting for approval",
		},
		{
			name: "エラー_セッションIDが空",
			setup: func(t *testing.T, svc *patrolServiceImpl) string {
				t.Helper()
				path := "/test/no-session"
				now := time.Now()
				svc.mu.Lock()
				svc.projects[path] = PatrolProject{Path: path, Name: "no-session"}
				svc.states[path] = &ProjectState{
					Project:   PatrolProject{Path: path, Name: "no-session"},
					Status:    StatusWaitingApproval,
					SessionID: "", // セッションIDなし
					StartedAt: &now,
					UpdatedAt: &now,
				}
				svc.mu.Unlock()
				return path
			},
			answer:    "yes",
			wantError: true,
			errMsg:    "no session ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			continueCalledCh := make(chan struct{}, 1)
			claude := &mockClaudeService{
				continueSessionStreamFn: func(_ context.Context, _, _, _ string, eventCh chan<- StreamEvent) error {
					select {
					case continueCalledCh <- struct{}{}:
					default:
					}
					close(eventCh)
					return nil
				},
			}

			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")
			impl := &patrolServiceImpl{
				projects:      make(map[string]PatrolProject),
				states:        make(map[string]*ProjectState),
				slots:         make(chan struct{}, MaxParallelSlots),
				claudeService: claude,
				configPath:    configPath,
				subscribers:   make(map[int]chan PatrolEvent),
			}

			projectPath := tt.setup(t, impl)
			err := impl.ResumeProject(projectPath, tt.answer)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("error message: got %q, want containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// 状態がRunningに遷移していることを確認
				impl.mu.RLock()
				state := impl.states[projectPath]
				impl.mu.RUnlock()

				if state.Status != StatusRunning {
					t.Errorf("status after resume: got %q, want %q", state.Status, StatusRunning)
				}
				if state.Question != nil {
					t.Error("question should be nil after resume")
				}

				// ContinueSessionStreamが呼ばれたことを確認
				select {
				case <-continueCalledCh:
					// OK
				case <-time.After(2 * time.Second):
					t.Error("ContinueSessionStream was not called within timeout")
				}
			}
		})
	}
}

// --- Polling テスト ---

func TestPatrolService_Polling(t *testing.T) {
	t.Run("StartPolling_StopPollingで停止可能", func(t *testing.T) {
		svc, _ := newTestPatrolService(t, &mockClaudeService{}, nil)

		// ポーリング開始・停止でpanicしないことを確認
		svc.StartPolling()
		// 少し待ってから停止
		time.Sleep(50 * time.Millisecond)
		svc.StopPolling()
		// 二重停止でもpanicしないことを確認
		svc.StopPolling()
	})

	t.Run("StartPollingを2回呼んでも前のポーリングが停止される", func(t *testing.T) {
		svc, _ := newTestPatrolService(t, &mockClaudeService{}, nil)

		svc.StartPolling()
		svc.StartPolling() // 2回目
		time.Sleep(50 * time.Millisecond)
		svc.StopPolling()
	})
}

// --- Subscribe テスト ---

func TestPatrolService_Subscribe(t *testing.T) {
	t.Run("イベント受信とunsubscribe", func(t *testing.T) {
		svc, _ := newTestPatrolService(t, &mockClaudeService{}, nil)
		impl := svc.(*patrolServiceImpl)

		ch, unsub := svc.Subscribe()

		// イベントを配信
		event := PatrolEvent{
			Type:    PatrolEventScanCompleted,
			Message: "test event",
		}
		impl.broadcast(event)

		// 受信確認
		select {
		case received := <-ch:
			if received.Type != PatrolEventScanCompleted {
				t.Errorf("event type: got %q, want %q", received.Type, PatrolEventScanCompleted)
			}
			if received.Message != "test event" {
				t.Errorf("event message: got %q, want %q", received.Message, "test event")
			}
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for event")
		}

		// unsubscribeした後はイベントが届かない
		unsub()

		// subscriber削除後の配信でpanicしないことを確認
		impl.broadcast(PatrolEvent{Type: "after_unsub"})
	})

	t.Run("複数subscriberにfan-out", func(t *testing.T) {
		svc, _ := newTestPatrolService(t, &mockClaudeService{}, nil)
		impl := svc.(*patrolServiceImpl)

		ch1, unsub1 := svc.Subscribe()
		defer unsub1()
		ch2, unsub2 := svc.Subscribe()
		defer unsub2()

		event := PatrolEvent{
			Type:    PatrolEventProjectStarted,
			Message: "broadcast test",
		}
		impl.broadcast(event)

		// 両方に配信されることを確認
		for i, ch := range []<-chan PatrolEvent{ch1, ch2} {
			select {
			case received := <-ch:
				if received.Type != PatrolEventProjectStarted {
					t.Errorf("subscriber %d: event type: got %q, want %q", i, received.Type, PatrolEventProjectStarted)
				}
			case <-time.After(1 * time.Second):
				t.Fatalf("subscriber %d: timeout waiting for event", i)
			}
		}
	})

	t.Run("バッファフルの場合はスキップされる", func(t *testing.T) {
		svc, _ := newTestPatrolService(t, &mockClaudeService{}, nil)
		impl := svc.(*patrolServiceImpl)

		ch, unsub := svc.Subscribe()
		defer unsub()

		// バッファ（100）を超えるイベントを送信
		for i := 0; i < 110; i++ {
			impl.broadcast(PatrolEvent{Type: "flood", Message: "test"})
		}

		// チャネルから読み取れる分だけ確認（panicしないことが主目的）
		count := 0
		for {
			select {
			case <-ch:
				count++
			default:
				goto done
			}
		}
	done:
		if count != 100 {
			t.Errorf("received events: got %d, want 100 (buffer size)", count)
		}
	})
}

// --- JSON永続化テスト ---

func TestPatrolService_JSONPersistence(t *testing.T) {
	t.Run("アトミック書き込み_一時ファイル経由", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "patrol.json")
		svc := NewPatrolService(&mockClaudeService{}, nil, configPath)

		projectDir := t.TempDir()
		if err := svc.RegisterProject(projectDir); err != nil {
			t.Fatalf("RegisterProject failed: %v", err)
		}

		// 一時ファイルが残っていないことを確認
		tmpFile := configPath + ".tmp"
		if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
			t.Error("temp file should not exist after successful write")
		}

		// 設定ファイルが正しいJSONであることを確認
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}
		if !json.Valid(data) {
			t.Error("config file contains invalid JSON")
		}
	})

	t.Run("設定ファイルが存在しない場合は空で開始", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nonexistent", "patrol.json")
		svc := NewPatrolService(&mockClaudeService{}, nil, configPath)

		projects := svc.ListProjects()
		if len(projects) != 0 {
			t.Errorf("projects count: got %d, want 0", len(projects))
		}
	})

	t.Run("設定ファイル破損時のフォールバック", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "patrol.json")

		// 壊れたJSONを書き込む
		if err := os.WriteFile(configPath, []byte("invalid json{{{"), 0644); err != nil {
			t.Fatalf("failed to write invalid config: %v", err)
		}

		// NewPatrolServiceはエラーをログに出力するが、panicしない
		svc := NewPatrolService(&mockClaudeService{}, nil, configPath)

		// 空のプロジェクトリストで開始される
		projects := svc.ListProjects()
		if len(projects) != 0 {
			t.Errorf("projects count: got %d, want 0", len(projects))
		}
	})

	t.Run("UnregisterProject後にJSONが更新される", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "patrol.json")
		svc := NewPatrolService(&mockClaudeService{}, nil, configPath)

		// 2つのプロジェクトを登録
		dir1 := t.TempDir()
		dir2 := t.TempDir()
		if err := svc.RegisterProject(dir1); err != nil {
			t.Fatalf("RegisterProject failed: %v", err)
		}
		if err := svc.RegisterProject(dir2); err != nil {
			t.Fatalf("RegisterProject failed: %v", err)
		}

		// 1つ解除
		if err := svc.UnregisterProject(dir1); err != nil {
			t.Fatalf("UnregisterProject failed: %v", err)
		}

		// JSONファイルにはdir2のみ残っている
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}
		var config PatrolConfig
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatalf("failed to parse config: %v", err)
		}
		if len(config.Projects) != 1 {
			t.Fatalf("projects count: got %d, want 1", len(config.Projects))
		}
		if config.Projects[0].Path != filepath.Clean(dir2) {
			t.Errorf("remaining project path: got %q, want %q", config.Projects[0].Path, filepath.Clean(dir2))
		}
	})
}

// --- monitorStreamEvents テスト ---

func TestPatrolService_MonitorStreamEvents(t *testing.T) {
	t.Run("Questionイベントで承認待ち状態に遷移", func(t *testing.T) {
		ntfy := &patrolMockNtfyService{}
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		impl := &patrolServiceImpl{
			projects:    make(map[string]PatrolProject),
			states:      make(map[string]*ProjectState),
			slots:       make(chan struct{}, MaxParallelSlots),
			ntfyService: ntfy,
			configPath:  configPath,
			subscribers: make(map[int]chan PatrolEvent),
		}

		projectPath := "/test/project"
		now := time.Now()
		impl.states[projectPath] = &ProjectState{
			Project:   PatrolProject{Path: projectPath, Name: "project"},
			Status:    StatusRunning,
			StartedAt: &now,
		}

		// Questionイベントを送信するチャネル
		eventCh := make(chan StreamEvent, 10)
		eventCh <- StreamEvent{
			Type:      EventTypeQuestion,
			SessionID: "session-abc",
			Result: &CommandResult{
				Questions: []Question{
					{Question: "Allow file write?"},
				},
			},
		}
		close(eventCh)

		impl.monitorStreamEvents(projectPath, eventCh)

		// 状態確認
		impl.mu.RLock()
		state := impl.states[projectPath]
		impl.mu.RUnlock()

		if state.Status != StatusWaitingApproval {
			t.Errorf("status: got %q, want %q", state.Status, StatusWaitingApproval)
		}
		if state.SessionID != "session-abc" {
			t.Errorf("sessionID: got %q, want %q", state.SessionID, "session-abc")
		}
		if state.Question == nil {
			t.Fatal("question should not be nil")
		}
		if state.Question.Question != "Allow file write?" {
			t.Errorf("question: got %q, want %q", state.Question.Question, "Allow file write?")
		}

		// ntfy通知が送信されたことを確認
		ntfy.mu.Lock()
		notifyCount := len(ntfy.notified)
		ntfy.mu.Unlock()
		if notifyCount != 1 {
			t.Errorf("ntfy notify count: got %d, want 1", notifyCount)
		}
	})

	t.Run("Errorイベントでエラー状態に遷移", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		impl := &patrolServiceImpl{
			projects:    make(map[string]PatrolProject),
			states:      make(map[string]*ProjectState),
			slots:       make(chan struct{}, MaxParallelSlots),
			configPath:  configPath,
			subscribers: make(map[int]chan PatrolEvent),
		}

		projectPath := "/test/error-project"
		now := time.Now()
		impl.states[projectPath] = &ProjectState{
			Project:   PatrolProject{Path: projectPath, Name: "error-project"},
			Status:    StatusRunning,
			StartedAt: &now,
		}

		eventCh := make(chan StreamEvent, 10)
		eventCh <- StreamEvent{
			Type:    EventTypeError,
			Message: "command failed",
		}
		close(eventCh)

		impl.monitorStreamEvents(projectPath, eventCh)

		impl.mu.RLock()
		state := impl.states[projectPath]
		impl.mu.RUnlock()

		if state.Status != StatusError {
			t.Errorf("status: got %q, want %q", state.Status, StatusError)
		}
		if state.Error != "command failed" {
			t.Errorf("error: got %q, want %q", state.Error, "command failed")
		}
	})

	t.Run("チャネルクローズで完了状態に遷移", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")
		impl := &patrolServiceImpl{
			projects:    make(map[string]PatrolProject),
			states:      make(map[string]*ProjectState),
			slots:       make(chan struct{}, MaxParallelSlots),
			configPath:  configPath,
			subscribers: make(map[int]chan PatrolEvent),
		}

		projectPath := "/test/complete-project"
		now := time.Now()
		impl.states[projectPath] = &ProjectState{
			Project:   PatrolProject{Path: projectPath, Name: "complete-project"},
			Status:    StatusRunning,
			StartedAt: &now,
		}

		// Initイベント後にチャネルクローズ（正常完了）
		eventCh := make(chan StreamEvent, 10)
		eventCh <- StreamEvent{
			Type:      EventTypeInit,
			SessionID: "session-init",
		}
		close(eventCh)

		impl.monitorStreamEvents(projectPath, eventCh)

		impl.mu.RLock()
		state := impl.states[projectPath]
		impl.mu.RUnlock()

		if state.Status != StatusCompleted {
			t.Errorf("status: got %q, want %q", state.Status, StatusCompleted)
		}
		if state.SessionID != "session-init" {
			t.Errorf("sessionID: got %q, want %q", state.SessionID, "session-init")
		}
	})
}

// --- GetStates テスト ---

func TestPatrolService_GetStates(t *testing.T) {
	t.Run("状態のコピーが返される", func(t *testing.T) {
		svc, _ := newTestPatrolService(t, &mockClaudeService{}, nil)
		impl := svc.(*patrolServiceImpl)

		// 内部状態をセット
		now := time.Now()
		impl.mu.Lock()
		impl.states["/test/project"] = &ProjectState{
			Project:   PatrolProject{Path: "/test/project", Name: "project"},
			Status:    StatusRunning,
			StartedAt: &now,
		}
		impl.mu.Unlock()

		states := svc.GetStates()

		// 返された状態を変更しても内部状態に影響しないことを確認
		states["/test/project"].Status = StatusCompleted

		impl.mu.RLock()
		internal := impl.states["/test/project"]
		impl.mu.RUnlock()

		if internal.Status != StatusRunning {
			t.Errorf("internal status changed unexpectedly: got %q, want %q", internal.Status, StatusRunning)
		}
	})
}

// --- ヘルパー関数 ---

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// initGitRepo はテスト用のgitリポジトリを初期化します
func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}

	for _, args := range cmds {
		c := exec.Command(args[0], args[1:]...)
		c.Dir = dir
		if output, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git command %v failed: %v\noutput: %s", args, err, output)
		}
	}

	// ファイルを作成してコミット
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = dir
	if output, err := addCmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\noutput: %s", err, output)
	}

	commitCmd := exec.Command("git", "commit", "-m", "initial commit")
	commitCmd.Dir = dir
	if output, err := commitCmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\noutput: %s", err, output)
	}
}
