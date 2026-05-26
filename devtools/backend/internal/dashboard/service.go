package dashboard

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"ghostrunner/backend/internal/projects"
)

// Service はダッシュボードの状態集約と回答書き戻しを提供します
type Service interface {
	// GetState は全プロジェクトの集約状態を返します
	GetState(ctx context.Context) (State, error)
	// Answer は確認事項に回答を書き戻します
	Answer(ctx context.Context, req AnswerRequest) error
}

type serviceImpl struct {
	configPath      string
	ghostrunnerRoot string
	now             func() time.Time
}

// NewService は新しいServiceを生成します
func NewService(configPath, ghostrunnerRoot string) Service {
	return &serviceImpl{
		configPath:      configPath,
		ghostrunnerRoot: ghostrunnerRoot,
		now:             time.Now,
	}
}

// NewServiceWithClock はclock注入付きのServiceを生成します（テスト用）
func NewServiceWithClock(configPath, ghostrunnerRoot string, now func() time.Time) Service {
	return &serviceImpl{
		configPath:      configPath,
		ghostrunnerRoot: ghostrunnerRoot,
		now:             now,
	}
}

// GetState は全プロジェクトの集約状態を返します
func (s *serviceImpl) GetState(ctx context.Context) (State, error) {
	log.Printf("[DashboardService] GetState started")

	projs, err := projects.LoadProjects(s.configPath)
	if err != nil {
		return State{}, fmt.Errorf("failed to load projects: %w", err)
	}

	if projs == nil {
		log.Printf("[DashboardService] No projects config found, returning empty state")
		return State{
			Projects:    []ProjectState{},
			GeneratedAt: s.now().Format(time.RFC3339),
		}, nil
	}

	var states []ProjectState
	for _, p := range projs {
		select {
		case <-ctx.Done():
			return State{}, ctx.Err()
		default:
		}

		ps, err := ScanProject(p.Path, s.ghostrunnerRoot, s.now())
		if err != nil {
			log.Printf("[DashboardService] ScanProject failed: path=%s, error=%v", p.Path, err)
			// エラーでもwarning付きで含める
			ps = ProjectState{
				Name:       p.Name,
				Path:       p.Path,
				Attention:  AttentionWatching,
				Unanswered: []UnansweredQuestion{},
				Ops:        []OpsEntry{},
				Warnings:   []string{fmt.Sprintf("scan failed: %v", err)},
			}
		}
		states = append(states, ps)
	}

	// ソート: attention優先度ASC, isSelf ASC, name ASC（安定ソート）
	sort.SliceStable(states, func(i, j int) bool {
		pi := attentionPriority(states[i].Attention)
		pj := attentionPriority(states[j].Attention)
		if pi != pj {
			return pi < pj
		}
		if states[i].IsSelf != states[j].IsSelf {
			return !states[i].IsSelf // isSelf=falseが先
		}
		return states[i].Name < states[j].Name
	})

	result := State{
		Projects:    states,
		GeneratedAt: s.now().Format(time.RFC3339),
	}

	log.Printf("[DashboardService] GetState completed: projects=%d", len(states))
	return result, nil
}

// Answer は確認事項に回答を書き戻します
func (s *serviceImpl) Answer(ctx context.Context, req AnswerRequest) error {
	log.Printf("[DashboardService] Answer started: project=%s, plan=%s, line=%d", req.ProjectPath, req.PlanPath, req.LineStart)

	projs, err := projects.LoadProjects(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	if projs == nil {
		return fmt.Errorf("%w: no projects configured", ErrValidation)
	}

	if err := AnswerQuestion(req, projs); err != nil {
		log.Printf("[DashboardService] Answer failed: error=%v", err)
		return err
	}

	log.Printf("[DashboardService] Answer completed: project=%s, plan=%s", req.ProjectPath, req.PlanPath)
	return nil
}

// attentionPriority はAttentionのソート優先度を返します（小さいほど優先）
func attentionPriority(a Attention) int {
	switch a {
	case AttentionRequired:
		return 0
	case AttentionProgress:
		return 1
	case AttentionWatching:
		return 2
	default:
		return 3
	}
}
