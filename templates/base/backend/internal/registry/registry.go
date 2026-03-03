package registry

import "github.com/gin-gonic/gin"

// InitFunc はサーバー起動時に実行される初期化関数。
type InitFunc func() error

// RouteFunc は API ルートを登録する関数。
type RouteFunc func(api *gin.RouterGroup)

// CleanupFunc はサーバー終了時に実行されるクリーンアップ関数。
type CleanupFunc func()

var (
	initFuncs    []InitFunc
	routeFuncs   []RouteFunc
	cleanupFuncs []CleanupFunc
)

// OnInit は初期化関数を登録する。
func OnInit(f InitFunc) {
	initFuncs = append(initFuncs, f)
}

// OnRoute はルート登録関数を登録する。
func OnRoute(f RouteFunc) {
	routeFuncs = append(routeFuncs, f)
}

// OnCleanup はクリーンアップ関数を登録する。
func OnCleanup(f CleanupFunc) {
	cleanupFuncs = append(cleanupFuncs, f)
}

// RunInit は登録された全初期化関数を実行する。
func RunInit() error {
	for _, f := range initFuncs {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

// SetupRoutes は登録された全ルートを API グループに追加する。
func SetupRoutes(api *gin.RouterGroup) {
	for _, f := range routeFuncs {
		f(api)
	}
}

// RunCleanup は登録された全クリーンアップ関数を実行する。
func RunCleanup() {
	for _, f := range cleanupFuncs {
		f()
	}
}
