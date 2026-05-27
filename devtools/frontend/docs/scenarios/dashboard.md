# ユーザー操作シナリオ: 統括ダッシュボード

## 1. 全プロジェクトの状況を横断把握する

### 目的

全プロジェクトの開発進捗（カンバン）と運用状態を一画面で横断的に確認する。

### 前提条件

- devtools バックエンドが起動している
- 統括対象プロジェクトが `patrol_projects.json` に登録されている

### 操作フロー

1. `/dashboard` にアクセスする
   - トップ `/` のヘッダ右側にある blue 系「統括」ボタン（`title="統括ダッシュボード"`）からも遷移できる
   - 旧 `/patrol` URL に直接アクセスした場合も 308 リダイレクトで `/dashboard` に到達する
2. ページ表示と同時にダッシュボードデータが自動取得される
3. 各プロジェクトのカードが表示される
   - 開発カンバン件数（レビュー・待ち・実行中・完了）
   - 運用エントリ（進捗、本日実績、stale 検知、連続エラー）
   - 未回答質問の件数
4. カード左端のアテンションバーで注意度を確認する
   - 赤: 対応が必要
   - 黄: 未回答質問あり
   - 青: 進行中
   - グレー: 監視中

### 成功時

- 全プロジェクトの状態がカード一覧で表示される
- 15秒間隔で自動更新される（ポーリング ON 時）

### エラー時

- 画面上部にエラーメッセージが表示される（赤背景）
- 手動で「状況は？」ボタンを押して再取得できる

---

## 2. 「状況は？」ボタンで即時把握する

### 目的

ワンクリックで全プロジェクトの最新状態を取得し、even-terminal にも状況レポートを依頼する。

### 前提条件

- even-terminal が起動している
- チャットセッションが確立されている

### 操作フロー

1. ヘッダーの「状況は？」ボタンをクリックする
2. ダッシュボードデータが即時リフレッシュされる
3. 同時に even-terminal に「状況は？」とプロンプトが送信される
4. ChatTranscript にストリーミングで応答テキストが表示される
5. TTS ON の場合、応答完了時に音声で読み上げられる

### 成功時

- プロジェクトカードが最新状態に更新される
- ChatTranscript に状況レポートが表示される
- TTS ON であれば音声読み上げが行われる

### エラー時

- セッションが無効な場合、自動的にセッション一覧を再取得してリトライする
- リトライも失敗した場合、エラーメッセージが表示される

---

## 3. チャットで指示・問い合わせを行う

### 目的

even-terminal 経由で任意のテキスト指示を送信し、応答を受け取る。

### 前提条件

- even-terminal が起動している
- チャットセッションが確立されている

### 操作フロー

1. 画面下部の ChatInput にテキストを入力する
2. Enter キーを押す（または「送信」ボタンをクリック）
   - Shift+Enter で改行
3. ChatTranscript に「応答中...」と表示される
4. SSE 経由でストリーミングテキストが蓄積表示される
5. 応答完了で status が idle に戻る

### 成功時

- ChatTranscript に応答テキストが表示される
- TTS ON であれば音声で読み上げられる

### エラー時

- セッション無効: 自動リトライ1回
- SSE 接続断: 指数バックオフで最大10回再接続
- 再接続上限超過: 「SSE 接続に失敗しました（再接続上限）」と表示

---

## 4. 未回答質問に回答する

### 目的

プロジェクトの計画書にある確認事項（未回答質問）に対して回答を書き戻す。

### 前提条件

- 未回答質問が存在するプロジェクトがある

### 操作フロー

1. DashboardCard に「未回答: N件」と表示されている質問をクリックする
2. 回答フォームが展開される
3. テキストで回答を入力し、送信する
4. `POST /api/dashboard/answer` で計画書に回答が書き戻される
5. ダッシュボードが自動リフレッシュされる

### 成功時

- 回答が計画書に書き戻され、未回答件数が減少する
- ダッシュボードのカード表示が更新される

### エラー時

- alert でエラーメッセージが表示される

---

## 5. ポーリングを切り替える

### 目的

ダッシュボードの自動更新（15秒間隔のポーリング）を ON/OFF する。

### 操作フロー

1. ヘッダーのポーリングトグルボタンをクリックする
2. ON 時: 「自動更新」（緑背景）と表示、15秒間隔で自動取得
3. OFF 時: 「手動更新（聞いたら返す）」（白背景）と表示、自動取得を停止

### 状態の永続化

- localStorage に保存され、ページ再読み込み後も維持される

---

## 6. TTS（音声読み上げ）を切り替える

### 目的

チャット応答完了時の音声読み上げを ON/OFF する。主経路は ElevenLabs のサーバー TTS（Romaco の声）、失敗時は自動で Web Speech にフォールバックする。

### 前提条件

- バックエンド `/api/tts` が稼働している（ElevenLabs API キーは backend env で設定）
- フォールバック経路が必要な場合はブラウザが Web Speech API（SpeechSynthesis）に対応している

### 操作フロー

1. ヘッダーの TTS トグルボタンをクリックする
2. ON 時: 「TTS ON」（青背景）と表示。同時に `prime()` が走り、iOS Safari の autoplay 制約が解除される
3. OFF 時: 「TTS OFF」（白背景）と表示。進行中の fetch / 再生 / フォールバックを全て停止
4. TTS ON の状態でチャット応答が完了すると、Romaco の声で読み上げが行われる

### 状態の永続化

- localStorage（`ghostrunner_tts_enabled`）に保存され、ページ再読み込み後も維持される

---

## 6-A. 応答音声読み上げ（ElevenLabs 主経路）

### 目的

チャット応答を ElevenLabs の Romaco 声で AirPods 等から再生する。

### 前提条件

- TTS トグル ON
- バックエンド `/api/tts` が ElevenLabs に到達できる（API キー設定済み、ネットワーク疎通あり）

### 操作フロー

1. ChatInput でテキストを入力 → 送信ボタン or Enter
   - 送信タップの同期スコープで `tts.prime()` が走り、`<audio>` 要素が unlock される
2. SSE 経由でストリーミング応答が蓄積される
3. 応答完了（`result` イベント / `status:idle` / 3秒無音タイムアウトのいずれか）で `tts.speak(fullText)` が呼ばれる
4. `POST /api/tts` に `{ text }` を JSON 送信
5. 数秒以内に `audio/mpeg`（`audio/*`）の Blob レスポンスが返る
6. `URL.createObjectURL(blob)` で `<audio>.src` に紐付け、`audio.play()` を呼ぶ
7. `<audio>.onplaying`（実再生開始）で `isSpeaking=true`、`error` が `null` にクリアされる
8. AirPods / Bluetooth デバイスから Romaco の声が再生される
9. `<audio>.onended` で `isSpeaking=false` に戻り、Blob URL が revoke される

### 成功時

- AirPods / Bluetooth スピーカーから Romaco の声で応答が読み上げられる
- TopError バナーが消える（前回フォールバックが残っていた場合）

### 補足

- 再生中に次の `speak()` が来た場合、`stopAll` で進行中の fetch を abort、`<audio>` を pause、Blob URL を revoke してから新規 fetch を開始する
- セッション切替（`onSessionSwitch`）でも `tts.cancel()` が走る

---

## 6-B. ElevenLabs 失敗時の Web Speech フォールバック

### 目的

ElevenLabs 経路の失敗（API キー未設定 / レート超過 / ネットワーク失敗 / レスポンス不正 / autoplay block 等）に対して、無音にならずに Web Speech で読み上げを続ける。

### 発火条件（7 経路）

| 経路 | 発火タイミング |
|-----|-------------|
| fetch failure | ネットワーク失敗、CORS、DNS 解決失敗等（`AbortError` は除く） |
| HTTP 4xx-5xx | `response.ok === false`（401 認証失敗、429 レート超過、5xx サーバーエラー等） |
| Content-Type 欠落 | レスポンスに `Content-Type` ヘッダがない |
| Content-Type 不正 | `audio/` で始まらない（例: `application/json` のエラーレスポンス） |
| 空 Blob | `blob.size === 0` |
| `<audio>.onerror` | デコード失敗、不正な Blob 等 |
| `audio.play()` reject | autoplay block、ユーザージェスチャ未取得状態など |

### 動作フロー（自動）

1. ElevenLabs 経路のいずれかで失敗が検知される
2. `error` に `"ElevenLabs 接続失敗。Web Speech に降格しました"` がセットされる
3. TopError バナー（赤背景）に上記メッセージが表示される
4. `speakWithWebSpeech(text, ...)` で Web Speech 経由の読み上げに切り替わる
5. `utterance.onstart` で `isSpeaking=true`、`onend` で `isSpeaking=false` に戻る

### 既知制約

- iOS Safari の SpeechSynthesis は**内蔵スピーカー固定**で、AirPods / Bluetooth デバイスには乗らない
- ElevenLabs 主経路（`<audio>` 再生）は OS の出力ルーティングに従うので、Romaco の声は AirPods 等にも乗る
- つまりフォールバック中だけ「音が内蔵スピーカーから出る」現象が発生する

### エラー復旧

- 次回 `speak()` で ElevenLabs 経路の再生が成功し、`<audio>.onplaying` が発火したタイミングで `error` が `null` にクリアされる
- `play` メソッドの resolve ではなく、実再生開始イベント（`playing`）で判定するため、autoplay block 等で「`play()` は resolve したが音は出ていない」状態を誤って成功扱いしない

### 失敗時のフェールセーフ

- Web Speech 経路でもエラーが起きた場合（`utterance.onerror`）、`isSpeaking=false` に戻すだけで追加通知はしない
- ブラウザが Web Speech にも対応していない環境では `speakWithWebSpeech` が no-op になり、音は出ないが UI はフリーズしない

---

## 7. チャットセッションを切り替える

### 目的

過去の対話セッションに戻る、または別の session で会話を続ける。

### 前提条件

- even-terminal に複数のセッションが存在する
- dashboard ヘッダの SessionPicker が表示されている

### 操作フロー

1. ヘッダ左の SessionPicker をタップしてドロップダウンを開く
   - 展開時に最新の session 一覧を再取得する
2. 一覧から切り替えたい session を選ぶ（`title` または `id` 先頭 8 文字 + 相対時刻 + status）
3. 既存 SSE 接続が close され、新しい session の SSE が開く
4. TTS が読み上げ中だった場合は自動で cancel される（`onSessionSwitch: tts.cancel`）
5. localStorage の `ghostrunner_active_session_id` が更新され、次回ロード時にも反映される

### 成功時

- ChatTranscript が新しい session のコンテキストにリセットされる
- ConnectionIndicator が live になる

### エラー時

- 切替先 session が無効（SSE が開かない）場合、指数バックオフで再接続を試行
- 10 回失敗で ConnectionIndicator が offline に

---

## 8. 新規セッションを開始する

### 目的

過去の文脈を引きずらずに、まっさらな session で会話を始める。

### 操作フロー

1. SessionPicker のドロップダウンを開く
2. 最上部の「+ 新規 session」をタップ
3. sessionId が null になり、localStorage からも削除される
4. ChatInput でメッセージを送ると、sessionId 省略で `POST /api/prompt` が叩かれる
5. even-terminal が新しい SID を発行し、レスポンス body の `{ sessionId }` から取得
6. その新 SID で SSE 接続を開始し、以降は通常の対話フロー

### 成功時

- 新 SID が localStorage に保存される
- SessionPicker の一覧に新規 session が出現する（次回展開時）

### エラー時

- `POST /api/prompt` 失敗時は「送信に失敗しました」エラーを表示
- レスポンス body から sessionId が取れない場合は「新規セッションの発行に失敗しました」を表示

---

## 9. セッション ID を別端末に共有する（SID コピー）

### 目的

dashboard で開いている session を、VSCode の `claude --resume <SID>` で続行する。

### 前提条件

- 現在の sessionId が確立されている（`null` ではない）

### 操作フロー

1. SessionPicker で対象 session を選ぶ
2. 隣の「SID」ボタンをタップ
3. `navigator.clipboard.writeText(sessionId)` でコピー
4. ラベルが「Copied」に 2 秒間変化する
5. 別端末（VSCode 等）でターミナルを開き、`claude --resume <貼り付け>` を実行

### secure context が無い場合（HTTP）

- `navigator.clipboard` が使えない環境では `document.execCommand('copy')` にフォールバック
- それも失敗した場合、SID を読み取り専用 `<input>` で表示するので手動コピー
- 詳細: `devtools/frontend/docs/modality-guide.md`

### 書き込み排他

- 同一 session への書き込みは同時 1 経路のみ（運用ルール、コードロックなし）
- dashboard で発話中は VSCode 側で送信しない、逆も同様

---

## 10. iOS Safari で背景復帰したときの整合性（FE-17）

### 目的

スマホ Safari で dashboard を開いたまま別アプリ → 戻った瞬間に「応答が途中で消えた」現象を防ぐ。

### 前提条件

- iOS Safari で `/dashboard` を開いている
- `text_delta` がストリーミング中にホーム画面に戻る等で background になった

### 動作フロー（自動）

1. `visibilitychange` で `visible` に戻る
2. `useChat` が `getHistory(sessionId, 5)` を 1 回叩く
3. 取得したアシスタント応答テキストで ChatTranscript を上書き
4. 同時に SSE 接続を再開（ConnectionIndicator は reconnecting → live へ遷移）
5. TTS は呼ばない（iOS の autoplay 制約のため）

### 失敗時の挙動

- `getHistory` が失敗しても黙ってスキップ（`error` にはしない）
- SSE 再接続自体は通常通り継続
- 失敗してもユーザー操作は止まらない
