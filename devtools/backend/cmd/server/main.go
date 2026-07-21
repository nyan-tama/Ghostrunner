// Package main はGhostrunner APIサーバーのエントリーポイントです
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"ghostrunner/backend/internal/dashboard"
	"ghostrunner/backend/internal/handler"
	"ghostrunner/backend/internal/projects"
	"ghostrunner/backend/internal/service"
	"ghostrunner/backend/internal/transcript"
	"ghostrunner/backend/internal/tts"

	"github.com/gin-contrib/cors"
	ginpprof "github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("[Server] Starting Ghostrunner API server...")

	// 依存性の組み立て
	ntfyService := service.NewNtfyService() // nil の場合がある（NTFY_TOPIC 未設定時）
	claudeService := service.NewClaudeService(ntfyService)
	geminiService := service.NewGeminiService() // nil の場合がある（API キー未設定時）
	openaiService := service.NewOpenAIService() // nil の場合がある（API キー未設定時）
	// Ghostrunnerリポジトリルートを取得（devtools/backend/cmd/server/main.go から4階層上）
	_, thisFile, _, _ := runtime.Caller(0)
	ghostrunnerRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")

	// patrol_projects.json パス（複数ハンドラ/サービスで共用）
	patrolConfigPath := filepath.Join(ghostrunnerRoot, "devtools", "backend", "patrol_projects.json")

	planHandler := handler.NewPlanHandler(claudeService)
	commandHandler := handler.NewCommandHandler(claudeService, ghostrunnerRoot)
	geminiHandler := handler.NewGeminiHandler(geminiService)
	openaiHandler := handler.NewOpenAIHandler(openaiService)
	filesHandler := handler.NewFilesHandler()
	projectsHandler := handler.NewProjectsHandler(patrolConfigPath)
	healthHandler := handler.NewHealthHandler()

	// 巡回サービスの依存性組み立て
	patrolService := service.NewPatrolService(claudeService, ntfyService, patrolConfigPath)
	patrolHandler := handler.NewPatrolHandler(patrolService)

	// ホームディレクトリ解決（質問待ちマーカー・プロジェクト生成で共用）
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("[Server] Failed to get home directory: %v", err)
	}

	// ダッシュボードサービスの依存性組み立て（質問待ちを会話ログ直読みで検出）。
	// フックのマーカー方式は環境（VS Code 拡張）で AskUserQuestion を取りこぼすため、
	// 各セッションの会話ログ JSONL を直読みする transcriptReader を idle.Reader として注入する。
	idleReader := transcript.NewReader(homeDir, func() ([]projects.Project, error) {
		return projects.LoadProjects(patrolConfigPath)
	}, time.Now)
	dashboardService := dashboard.NewService(patrolConfigPath, ghostrunnerRoot, idleReader)

	// ダッシュボード状態のSSE配信サービス
	dashboardStream := dashboard.NewStreamService(dashboardService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService, dashboardStream)

	// バックグラウンドジョブを起動（サーバー稼働中は動き続ける）。
	// W3: 要約ジョブ（Summarizer）は Phase A では起動しない。transcript セッションには
	// マーカー(.idle)が無く、marker writer では WriteSummary が毎回破棄される一方、
	// その前に haiku を実行済みで永久 churn するため。要約は Phase B（要約キャッシュ writer）で有効化する。
	// SSE 配信（StreamService）は Phase A でも維持する。
	bgCtx := context.Background()
	dashboardStream.Start(bgCtx)

	// TTS (VOICEVOX) の依存性組み立て
	ttsService := tts.NewService()
	ttsHandler := tts.NewHandler(ttsService)

	// プロジェクト生成関連の依存性組み立て
	templateService := service.NewTemplateService(ghostrunnerRoot)
	createService := service.NewCreateService(templateService, homeDir)
	createHandler := handler.NewCreateHandler(createService)

	// Ginエンジン初期化
	r := gin.Default()

	// pprof エンドポイント (メモリリーク調査用)
	// ENABLE_PPROF=1 のときだけ有効化。本番事故防止のため明示 opt-in。
	// 取得例:
	//   go tool pprof http://localhost:8888/debug/pprof/heap
	//   curl -s http://localhost:8888/debug/pprof/heap > heap.pprof
	if os.Getenv("ENABLE_PPROF") == "1" {
		ginpprof.Register(r)
		log.Println("[Server] pprof enabled at /debug/pprof/")
	}

	// CORS設定（ローカル開発時およびTailscale経由のアクセスを許可）
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// localhost を許可（3000: プロジェクト用, 3333: devtools用）
			if origin == "http://localhost:3000" || origin == "http://localhost:3333" {
				return true
			}
			// Tailscale IP (100.x.x.x) を許可
			if len(origin) > 11 && origin[:11] == "http://100." {
				return true
			}
			// Tailscale Funnel ドメイン (*.ts.net) を許可
			if len(origin) > 7 && origin[len(origin)-7:] == ".ts.net" {
				return true
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	// APIルーティング
	api := r.Group("/api")
	{
		// ヘルスチェックAPI
		api.GET("/health", healthHandler.Handle)

		// ファイル一覧API
		api.GET("/files", filesHandler.Handle)

		// プロジェクト一覧API
		api.GET("/projects", projectsHandler.Handle)
		api.POST("/projects/destroy", projectsHandler.HandleDestroy)

		// 汎用コマンドAPI（推奨）
		api.POST("/command", commandHandler.Handle)
		api.POST("/command/stream", commandHandler.HandleStream)
		api.POST("/command/continue", commandHandler.HandleContinue)
		api.POST("/command/continue/stream", commandHandler.HandleContinueStream)

		// 旧API（互換性維持）
		api.POST("/plan", planHandler.Handle)
		api.POST("/plan/stream", planHandler.HandleStream)
		api.POST("/plan/continue", planHandler.HandleContinue)
		api.POST("/plan/continue/stream", planHandler.HandleContinueStream)

		// Gemini API
		api.POST("/gemini/token", geminiHandler.HandleToken)

		// OpenAI Realtime API
		api.POST("/openai/realtime/session", openaiHandler.HandleSession)

		// プロジェクト生成API
		api.GET("/projects/validate", createHandler.HandleValidate)
		api.POST("/projects/create/stream", createHandler.HandleCreateStream)
		api.POST("/projects/open", createHandler.HandleOpen)

		// TTS API (VOICEVOX)
		api.POST("/tts", ttsHandler.HandleSynthesize)

		// ダッシュボードAPI
		dashGroup := api.Group("/dashboard")
		{
			dashGroup.GET("/state", dashboardHandler.HandleState)
			dashGroup.POST("/answer", dashboardHandler.HandleAnswer)
			dashGroup.GET("/stream", dashboardHandler.HandleStream)
		}

		// 巡回API
		patrol := api.Group("/patrol")
		{
			patrol.POST("/projects", patrolHandler.HandleRegister)
			patrol.POST("/projects/remove", patrolHandler.HandleRemove)
			patrol.GET("/projects", patrolHandler.HandleListProjects)
			patrol.GET("/scan", patrolHandler.HandleScan)
			patrol.POST("/start", patrolHandler.HandleStart)
			patrol.POST("/stop", patrolHandler.HandleStop)
			patrol.POST("/resume", patrolHandler.HandleResume)
			patrol.GET("/states", patrolHandler.HandleStates)
			patrol.GET("/stream", patrolHandler.HandleStream)
			patrol.POST("/polling/start", patrolHandler.HandlePollingStart)
			patrol.POST("/polling/stop", patrolHandler.HandlePollingStop)
		}
	}

	// サーバー起動（0.0.0.0で全インターフェースからアクセス可能に）
	log.Println("[Server] Listening on 0.0.0.0:8888")
	if err := r.Run("0.0.0.0:8888"); err != nil {
		log.Fatalf("[Server] Failed to start server: %v", err)
	}
}
