# OpenAI Realtime API 実装概要

外部システムで音声対話機能を実装するための概要資料。

## 1. 構成概要

```
[ブラウザ] <--WebSocket--> [OpenAI Realtime API]
     |
     v (HTTP)
[バックエンド] --> エフェメラルトークン発行
```

**ポイント:**
- ブラウザから直接 OpenAI に WebSocket 接続
- バックエンドはトークン発行のみ（APIキーを隠蔽）

---

## 2. バックエンド設定

### 環境変数

```
OPENAI_API_KEY=sk-xxxxx
```

### エンドポイント

```
POST /api/openai/realtime/session
```

**リクエスト:**
```json
{
  "model": "gpt-realtime",  // オプション
  "voice": "verse"          // オプション
}
```

**レスポンス:**
```json
{
  "success": true,
  "token": "ek_xxxxx",
  "expireTime": "2026-02-02T12:00:00Z"
}
```

### トークン発行API（OpenAI）

```
POST https://api.openai.com/v1/realtime/client_secrets
Authorization: Bearer {OPENAI_API_KEY}

{
  "expires_after": {
    "anchor": "created_at",
    "seconds": 600
  },
  "session": {
    "type": "realtime",
    "model": "gpt-realtime",
    "audio": {
      "output": {
        "voice": "verse"
      }
    }
  }
}
```

---

## 3. フロントエンド設定

### WebSocket接続

```javascript
const wsUrl = "wss://api.openai.com/v1/realtime?model=gpt-realtime";
const ws = new WebSocket(wsUrl, [
  "realtime",
  `openai-insecure-api-key.${token}`  // サブプロトコルでトークンを渡す
]);
```

### 音声フォーマット

| 項目 | 値 |
|------|-----|
| 入力サンプルレート | 24000 Hz |
| 出力サンプルレート | 24000 Hz |
| フォーマット | PCM 16bit |
| エンコード | Base64 |

---

## 4. メッセージフロー

### 接続時

1. WebSocket 接続
2. サーバーから `session.created` 受信
3. クライアントから `session.update` 送信
4. サーバーから `session.updated` 受信 → 接続完了

### session.update の例

```json
{
  "type": "session.update",
  "session": {
    "type": "realtime",
    "instructions": "あなたは親切な音声アシスタントです。日本語で会話してください。",
    "audio": {
      "input": {
        "format": { "type": "audio/pcm", "rate": 24000 },
        "turn_detection": {
          "type": "server_vad",
          "threshold": 0.5,
          "prefix_padding_ms": 300,
          "silence_duration_ms": 500
        }
      },
      "output": {
        "format": { "type": "audio/pcm", "rate": 24000 },
        "voice": "verse"
      }
    }
  }
}
```

### 音声入力（マイク）

```json
{
  "type": "input_audio_buffer.append",
  "audio": "{base64_pcm_data}"
}
```

### テキスト入力

```json
// 1. メッセージを追加
{
  "type": "conversation.item.create",
  "item": {
    "type": "message",
    "role": "user",
    "content": [{ "type": "input_text", "text": "こんにちは" }]
  }
}

// 2. 応答を要求
{
  "type": "response.create"
}
```

### 音声出力受信

```json
{
  "type": "response.output_audio.delta",
  "delta": "{base64_pcm_data}",
  "response_id": "xxx",
  "item_id": "xxx"
}
```

---

## 5. 注意点

### トークン有効期限

- デフォルト: 600秒（10分）
- 期限切れで WebSocket 切断
- 自動再接続を実装すること

### 自動再接続

- 最大試行回数: 3回
- 指数バックオフ: 1秒 → 2秒 → 4秒
- 意図的な切断時は再接続しない

### 音声出力

- 複数の `response.output_audio.delta` が連続で届く
- キューに溜めて順次再生
- AudioContext は遅延作成（ユーザー操作後）

### マイク入力

- ネイティブサンプルレートから 24kHz にリサンプリング
- Float32 → Int16 → Base64 変換
- echoCancellation, noiseSuppression 推奨

### ブラウザ制約

- AudioContext はユーザー操作後に作成
- バックグラウンドタブでは音声再生制限あり

---

## 6. 利用可能な音声（voice）

| voice | 特徴 |
|-------|------|
| verse | デフォルト、自然な声 |
| alloy | 中性的 |
| echo | 男性的 |
| shimmer | 女性的 |

---

## 7. 主要イベント一覧

| イベント | 方向 | 説明 |
|---------|------|------|
| session.created | 受信 | 接続成功 |
| session.update | 送信 | セッション設定 |
| session.updated | 受信 | 設定完了 |
| input_audio_buffer.append | 送信 | 音声入力 |
| conversation.item.create | 送信 | テキスト入力 |
| response.create | 送信 | 応答要求 |
| response.output_audio.delta | 受信 | 音声出力（断片） |
| response.output_audio.done | 受信 | 音声出力完了 |
| error | 受信 | エラー |

---

## 8. 参考リンク

- [OpenAI Realtime API Documentation](https://platform.openai.com/docs/guides/realtime)
- [WebSocket API](https://platform.openai.com/docs/api-reference/realtime)
