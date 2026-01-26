# 検討: Gemini Live API 接続時間制限への対応

## 背景

Gemini Live API を使った音声UI実装において、以下の制限が課題となる:

| 制限項目 | 値 |
|---------|-----|
| 音声セッション最大時間 | 15分 |
| WebSocket接続寿命 | 約10分 |

### 実際の利用パターン

```
[音声指示 30秒] → [待機 5-30分] → [通知音声] → [確認/次の指示 30秒]
                     ↓
              Claude Code実行中
              エディタ作業中
```

**問題点:**
- 1サイクルで15分を超えることがある
- 待機中も接続を維持する必要がある（音声通知のため）
- 15分経過時に接続が切断され、通知を音声で伝えられない

---

## 採用方針: 会話履歴保存 + 新規セッション継続

### 選定理由

**候補1: セッション再開機能**
- 会話履歴が完全保持される（理想的）
- しかし実装が複雑
- 資料に「既知のバグ（途中切断、ハング）」の記載あり
- 安定性に不安

**候補2: 会話履歴保存 + 新規セッション（採用）**
- シンプルで確実
- Gemini側の不具合の影響を受けにくい
- 直近の文脈のみ保持すればトークン消費も抑えられる

---

## 実装方針

### 1. 基本的なアプローチ

14分経過時点で以下を実行:

1. 現在の会話履歴（直近5-10ターン）を保存
2. WebSocket接続を切断
3. 新規セッションを開始
4. システム指示に会話履歴を含めて送信

### 2. 会話履歴の保存範囲

**直近5-10ターンのみ保存（推奨）**

理由:
- 音声UIの特性上、長大な履歴は不要
- トークン消費を抑える
- システム指示が肥大化しない

保存対象:
```typescript
interface ConversationTurn {
  role: 'user' | 'model';
  content: string;
  timestamp: number;
  functionCalls?: FunctionCall[];  // ツール呼び出しも含める
}
```

### 3. 実装イメージ

```typescript
// frontend/src/hooks/useGeminiLive.ts

const MAX_HISTORY_TURNS = 10;
const SESSION_DURATION_MS = 14 * 60 * 1000; // 14分

let conversationHistory: ConversationTurn[] = [];
let sessionStartTime = Date.now();

// セッション時間監視
useEffect(() => {
  const timer = setInterval(() => {
    const elapsed = Date.now() - sessionStartTime;

    if (elapsed > SESSION_DURATION_MS) {
      reconnectWithHistory();
    }
  }, 10000); // 10秒ごとにチェック

  return () => clearInterval(timer);
}, []);

async function reconnectWithHistory() {
  // 1. 現在の音声を停止
  stopCurrentAudio();

  // 2. 接続を切断
  ws.close();

  // 3. 直近の履歴を取得
  const recentHistory = conversationHistory.slice(-MAX_HISTORY_TURNS);

  // 4. 新規セッションを開始
  const newConfig = {
    model: "gemini-2.5-flash-native-audio-preview-12-2025",
    systemInstruction: {
      parts: [
        { text: BASE_SYSTEM_INSTRUCTION },
        { text: buildHistoryContext(recentHistory) }
      ]
    },
    generationConfig: {
      responseModalities: ["AUDIO"]
    },
    tools: FUNCTION_DECLARATIONS
  };

  await connect(newConfig);

  sessionStartTime = Date.now();
}

function buildHistoryContext(history: ConversationTurn[]): string {
  return `
以前の会話履歴（直近${history.length}ターン）:

${history.map((turn, i) => `
${turn.role === 'user' ? 'ユーザー' : 'あなた'}: ${turn.content}
${turn.functionCalls ? `実行したツール: ${turn.functionCalls.map(fc => fc.name).join(', ')}` : ''}
`).join('\n')}

上記の文脈を踏まえて会話を継続してください。
`;
}

// メッセージ受信時に履歴を更新
function onMessage(data: ServerMessage) {
  if (data.serverContent?.modelTurn) {
    conversationHistory.push({
      role: 'model',
      content: extractTextFromTurn(data.serverContent.modelTurn),
      timestamp: Date.now(),
      functionCalls: data.serverContent.modelTurn.parts
        ?.filter(p => p.functionCall)
        .map(p => p.functionCall)
    });
  }
}

function onUserSpeech(transcript: string) {
  conversationHistory.push({
    role: 'user',
    content: transcript,
    timestamp: Date.now()
  });
}
```

### 4. UI/UX の考慮事項

**再接続時の動作:**
```typescript
// 14分経過時点で静かに再接続
// ユーザーへの通知なし（技術的制約なので気にしなくていい）

async function reconnectWithHistory() {
  // 1. 現在の音声を停止
  stopCurrentAudio();

  // 2. 接続を切断
  ws.close();

  // 3. 新規セッションを開始（通知なし）
  await connect(newConfig);

  sessionStartTime = Date.now();
}
```

**タスク完了時の通知:**
- Web Notification + 効果音でバックグラウンド通知
- ユーザーがクリックしてアプリにフォーカス後、Gemini Liveで詳細を音声説明
- 詳細: `開発/検討中/2026-01-26_バックグラウンド通知設計.md`

---

## トレードオフ

### メリット

1. **実装がシンプル**
   - セッション再開APIの複雑さを回避
   - タイマーと履歴管理のみ

2. **安定性が高い**
   - Gemini側の既知のバグの影響を受けにくい
   - 新規接続は最も確実

3. **トークン消費を抑制**
   - 直近履歴のみで十分な文脈を保持
   - システム指示の肥大化を防ぐ

### デメリット

1. **完全な文脈保持は不可**
   - 直近10ターン以前の情報は失われる
   - 長期的な文脈が必要な会話には不向き

2. **再接続の瞬間に遅延**
   - 1-2秒のダウンタイムが発生
   - 音声入力が一時的に途切れる

3. **Function Call履歴の再現が不完全**
   - 実行済みツールの結果は文字列として保存
   - モデルが再度同じツールを呼ぶ可能性

---

## 実装上の注意点

### 1. Function Call履歴の扱い

ツール呼び出しの結果も履歴に含める:

```typescript
interface ConversationTurn {
  role: 'user' | 'model';
  content: string;
  functionCalls?: {
    name: string;
    args: Record<string, any>;
    result: string;  // 実行結果を文字列化
  }[];
}
```

システム指示に含める形式:
```
モデル: 顧客情報を取得します
実行したツール: get_customer_info({ id: "12345" })
結果: { name: "田中太郎", status: "active" }
```

### 2. エラーハンドリング

```typescript
async function reconnectWithHistory() {
  try {
    await connect(newConfig);
  } catch (error) {
    // フォールバック: 履歴なしで接続
    console.error('Failed to reconnect with history:', error);
    await connect(defaultConfig);

    // ユーザーに通知
    playAudio("接続を再開しました。会話の文脈が一部失われている可能性があります。");
  }
}
```

### 3. 音声の途切れ対策

```typescript
// 再接続中のバッファリング
let audioQueue: AudioChunk[] = [];
let isReconnecting = false;

function onAudioInput(chunk: AudioChunk) {
  if (isReconnecting) {
    audioQueue.push(chunk);
  } else {
    sendAudioChunk(chunk);
  }
}

async function reconnectWithHistory() {
  isReconnecting = true;

  await connect(newConfig);

  // バッファした音声を送信
  while (audioQueue.length > 0) {
    const chunk = audioQueue.shift();
    sendAudioChunk(chunk);
  }

  isReconnecting = false;
}
```

---

## 次のステップ

1. **プロトタイプ実装**
   - `useGeminiLive` フックに履歴管理機能を追加
   - 14分タイマーと自動再接続を実装

2. **テスト**
   - 15分以上のセッションで動作確認
   - 再接続前後の文脈継続性をチェック
   - Function Call実行中の再接続ケースを検証

3. **本実装**
   - エラーハンドリングの強化
   - UI通知の追加
   - ログ記録と監視

---

## 参考資料

- `開発/資料/2026-01-25_gemini-live-function-calling.md` - セッション制限の詳細
- Gemini Live API ドキュメント - Session management
