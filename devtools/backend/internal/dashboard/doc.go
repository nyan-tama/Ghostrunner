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
//   - IdleState: 質問待ち状態（会話ログ由来の代表マーカー。キー存在＝質問待ち）
//   - RunningState: 動作中状態（会話ログ上で Claude が処理中の代表セッション。キー存在＝動作中。
//     kanban.running 件数・ops status="running" とは別概念のランタイム動作中）
//   - AnswerRequest: 回答書き戻しリクエスト（プロジェクトパス、計画書パス、行番号、回答文）
//
// # 主要な関数・インターフェース
//
//   - Service: GetState（全プロジェクト集約）とAnswer（回答書き戻し）を提供するインターフェース
//   - NewService: Serviceの本番用コンストラクタ
//   - NewServiceWithClock: clock注入付きコンストラクタ（テスト用）
//   - ScanProject: 1プロジェクトのカンバン/未回答/運用を読み取り専用で収集する
//   - AnswerQuestion: 計画書の未回答行を「回答済」に更新し回答文を挿入する（アトミック書き込み）
//   - StreamService: ダッシュボード状態のSSE配信（変化時のみStateスナップショットをbroadcast）
//   - Summarizer: 滞留した質問待ちマーカーを検出しSummarizeServiceで要約してマーカーへ書き戻す
//
// # 質問待ち要約とSSE配信（Phase 1b）
//
// Summarizerはtickerで滞留（約2分以上）かつ未要約のマーカーを抽出し、会話末尾を
// service.SummarizeService（claude -p --model haiku）で日本語1行に要約してidle.Writerで
// マーカーへ書き戻す。書き戻しはフックによる削除/更新を rename 直前に再確認し、解消済みの
// 質問待ちを復活させない（compare-and-swap）。並列数は小さく抑えCLIコストを制御する。
//
// StreamServiceは短間隔でGetStateをスキャンし、前回と実変化があった場合のみStateスナップ
// ショット全体をsubscriberへbroadcastする。generatedAtや経過時間は差分判定に含めず、projects
// の実変化のみをトリガーとする（時間経過だけでは再送しない）。subscriberチャネルは最新優先の
// 小バッファで、満杯時は古い値を捨てて最新を入れる（coalesce）。
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
//
// # 質問待ち(Idle)・動作中(Running)の集約
//
// GetStateは各プロジェクトのScanProject結果に対し、idleパッケージ(reader)が読み取った
// 代表マーカーを後段で付与する(attachIdleState)。idleReaderがnilの場合はこの付与を丸ごと
// スキップする。readerは各プロジェクトの最新mtimeセッション1件をStatus付き(waiting/running)で
// 返すため、dashboardは代表選定をせずStatusでディスパッチするのみ(C-1)。集約の要点は以下のとおり。
//
//   - しきい値(TTL=6時間・MinAge=60秒)はidleパッケージに集約(SSOT)。transcriptの分類しきい値と
//     重複させないため、dashboardもidleの定数を参照する
//   - TTL(idle.TTL)を超過した失効マーカーは除外する(実ファイルの削除はしない・読み取り専用)
//   - マーカーのcwdはidle.MatchProjectで登録済みプロジェクトへパス前方一致で紐付ける
//   - Marker.Statusで分岐: waiting→IdleState、running→RunningStateを付与。SessionCountは
//     rep.SessionCount(reader集計の同一status数)をそのまま採用する
//   - idleMinAgeゲート(応答直後ノイズ抑制)はwaitingのみに適用し、runningには適用しない(fresh runningを
//     落とすと動作中が一切表示されないため・C-1)
//   - 付与したプロジェクトはAttentionを再評価する(determineAttention・C1)。質問待ちはrequired、
//     動作中はrequired要因が無ければprogressになる
//   - プロジェクトのソートは質問待ち(Idle!=nil)を第1キー、動作中(Running!=nil)を第2キーとし、
//     未回答由来のrequiredより優先する。以降はattention優先度、質問待ちの経過時間(内部計算・非露出)、
//     isSelf、名前の順で安定ソートする
package dashboard
