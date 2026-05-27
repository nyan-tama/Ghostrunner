// Package dashboard は統括GUIダッシュボードの状態集約と回答書き戻しを提供する。
//
// # 概要
//
// 登録済みプロジェクト群のカンバン状態、未回答確認事項、運用状態を
// ファイルシステムから読み取り専用で集約する。
// 唯一の書き込み操作は確認事項への回答書き戻し(AnswerQuestion)のみ。
//
// # 主要な型
//
//   - State: ダッシュボード全体の集約結果（ProjectState配列とgeneratedAt）
//   - ProjectState: 1プロジェクトの状態（カンバン件数、未回答事項、運用状態、注目度）
//   - Attention: プロジェクトの注目度（required / progress / watching）
//   - KanbanCounts: カンバン各レーン（レビュー/実装待ち/実行中/完了）の.mdファイル数
//   - UnansweredQuestion: 計画書内の未回答確認事項（ファイルパス、行番号、質問文）
//   - OpsEntry: 運用/状態/ 配下のJSONから読み取った1エントリ（stale検知付き）
//   - AnswerRequest: 回答書き戻しリクエスト（プロジェクトパス、計画書パス、行番号、回答文）
//
// # 主要な関数・インターフェース
//
//   - Service: GetState（全プロジェクト集約）とAnswer（回答書き戻し）を提供するインターフェース
//   - NewService: Serviceの本番用コンストラクタ
//   - NewServiceWithClock: clock注入付きコンストラクタ（テスト用）
//   - ScanProject: 1プロジェクトのカンバン/未回答/運用を読み取り専用で収集する
//   - AnswerQuestion: 計画書の未回答行を「回答済」に更新し回答文を挿入する（アトミック書き込み）
//
// # 設計方針
//
//   - ファイルシステムを唯一の真実源(source of truth)とする
//   - 未回答検出の正規表現パターンはgrrunパッケージのSSOTを共有
//   - テスト用にclock注入(NewServiceWithClock)をサポート
//   - ScanProjectは各プロジェクトの状態を独立に収集し、エラーはwarningsに蓄積
//   - AnswerQuestionはwrite-to-temp + renameパターンで安全にファイルを更新
//   - 回答対象の計画書は開発/実装/実装待ち/ または 開発/実装/実行中/ 配下の.mdのみ許可
//   - プロジェクトパスはpatrol_projects.jsonの登録済みリストで検証
package dashboard
