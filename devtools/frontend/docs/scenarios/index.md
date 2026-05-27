# ユーザー操作シナリオ一覧

Ghost Runner の主要なユーザー操作シナリオを記載する。

## シナリオカテゴリ

| カテゴリ | ファイル | 内容 |
|---------|---------|------|
| コマンド実行 | [command-execution.md](./command-execution.md) | コマンドの入力と実行 |
| 音声通知 | [voice-notification.md](./voice-notification.md) | 音声通知機能の利用 |
| プロジェクト作成 | [project-create.md](./project-create.md) | プロジェクトの新規作成 |
| プロジェクト削除 | [project-delete.md](./project-delete.md) | プロジェクトの登録解除 |
| 巡回ダッシュボード（旧） | [patrol.md](./patrol.md) | 統括ダッシュボードに役割を移譲済み。`/patrol` は `/dashboard` へ 308 リダイレクトされる |
| 統括ダッシュボード | [dashboard.md](./dashboard.md) | 全プロジェクト横断把握・チャット・音声読み上げ。トップ `/` のヘッダ「統括」ボタンから遷移する |

## 基本フロー

1. プロジェクトをドロップダウンから選択（または履歴から選択）
2. コマンドを選択
3. （任意）ファイルを選択
4. 引数を入力
5. （任意）PR workflow を有効化
6. （任意）Voice notification を有効化
7. （任意）画像を添付（ファイル選択 / ドラッグ&ドロップ / カメラ撮影）
8. Execute Command ボタンをクリック
9. 結果を確認（音声通知有効時は音声でも通知）
