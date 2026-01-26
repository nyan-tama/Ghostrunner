# 調査レポート: Gemini Live APIで音声応答が返ってこない問題

## 概要

Gemini Live APIでWebSocket接続後に音声を送信してもサーバーからの応答がない問題を調査した結果、**メッセージフォーマットの誤り**と**入力サンプルレートの不一致**が主な原因と判明した。

## 背景

- Go バックエンドでエフェメラルトークンを発行（正常動作）
- Next.js フロントエンドでWebSocket接続（setupComplete受信まで正常）
- Safari で音声録音が動作（amplitude が実際の値を示す）
- 音声データを送信しているが、Geminiからの応答がない

## 調査結果

### 公式ドキュメント

#### WebSocket エンドポイント

エフェメラルトークンを使用する場合、`v1alpha`と`BidiGenerateContentConstrained`を使用する必要がある:

```
wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContentConstrained?access_token={token}
```

現在の実装はこのエンドポイントを正しく使用している。

#### 音声入力フォーマット要件

| 項目 | 入力 | 出力 |
|------|------|------|
| フォーマット | Raw 16-bit PCM | Raw 16-bit PCM |
| サンプルレート | **16 kHz** | 24 kHz |
| エンディアン | Little-endian | Little-endian |
| チャンネル | モノラル | モノラル |

#### realtimeInputメッセージ形式（重要）

公式ドキュメントによると、`mediaChunks`は**非推奨**となり、`audio`フィールドを直接使用することが推奨される:

```json
{
  "realtimeInput": {
    "audio": {
      "data": "<base64-encoded-audio>",
      "mimeType": "audio/pcm;rate=16000"
    }
  }
}
```

旧形式（非推奨だが動作可能）:
```json
{
  "realtimeInput": {
    "mediaChunks": [{
      "data": "<base64-encoded-audio>",
      "mimeType": "audio/pcm;rate=16000"
    }]
  }
}
```

### サンプルコード

#### 正しい音声送信の実装例（JavaScript）

```javascript
// AudioContext設定 - 入力は16kHzを使用
const inputAudioContext = new AudioContext({ sampleRate: 16000 });
const scriptProcessor = inputAudioContext.createScriptProcessor(4096, 1, 1);

scriptProcessor.onaudioprocess = (e) => {
  const inputData = e.inputBuffer.getChannelData(0);

  // Float32 を Int16 PCM に変換
  const int16Data = new Int16Array(inputData.length);
  for (let i = 0; i < inputData.length; i++) {
    int16Data[i] = Math.max(-32768, Math.min(32767, inputData[i] * 32768));
  }

  // Base64エンコード
  const base64Data = btoa(String.fromCharCode(...new Uint8Array(int16Data.buffer)));

  // Geminiに送信
  ws.send(JSON.stringify({
    realtimeInput: {
      audio: {
        data: base64Data,
        mimeType: "audio/pcm;rate=16000"
      }
    }
  }));
};
```

### VAD（Voice Activity Detection）の動作条件

- デフォルトで自動VADが有効
- **音声末尾に十分な無音がないと応答がトリガーされない場合がある**
- 手動VADモードに切り替えることで問題を回避可能

#### VAD設定オプション

```javascript
const setupMessage = {
  setup: {
    model: "models/gemini-2.5-flash-native-audio-preview-12-2025",
    generationConfig: {
      responseModalities: ["AUDIO"]
    },
    realtimeInputConfig: {
      automaticActivityDetection: {
        disabled: false,  // 自動VADを有効
        startOfSpeechSensitivity: "START_SENSITIVITY_LOW",
        endOfSpeechSensitivity: "END_SENSITIVITY_LOW",
        prefixPaddingMs: 20,
        silenceDurationMs: 100
      }
    }
  }
};
```

#### 手動VADモード

自動VADを無効にして手動制御する場合:

```javascript
// setup時
realtimeInputConfig: {
  automaticActivityDetection: {
    disabled: true
  }
}

// 音声送信時
ws.send(JSON.stringify({ realtimeInput: { activityStart: {} } }));
// 音声データを送信...
ws.send(JSON.stringify({ realtimeInput: { activityEnd: {} } }));
```

## 比較表

| 項目 | 現在の実装 | 正しい実装 |
|------|------------|------------|
| メッセージ形式 | `{ media: {...} }` | `{ realtimeInput: { audio: {...} } }` |
| サンプルレート | 24 kHz | **16 kHz** |
| mimeType | `audio/pcm` | `audio/pcm;rate=16000` |
| WebSocketエンドポイント | v1alpha + Constrained | v1alpha + Constrained (正しい) |

## 既知の問題・注意点

### GitHub Issues

- [Issue #408 - 特定の音声に応答がない](https://github.com/google-gemini/cookbook/issues/408): VADが音声末尾の無音不足で応答をトリガーしない問題。手動VADモードで解決可能。
- [Issue #619 - 音声出力が動作しない](https://github.com/googleapis/js-genai/issues/619): React実装で音声出力が動作しない問題。
- [Issue #821 - エフェメラルトークンが動作しない](https://github.com/google-gemini/cookbook/issues/821): v1alphaエンドポイントを使用する必要がある。

### よくあるエラーパターン

1. **メッセージフォーマットの誤り**: `media`ではなく`realtimeInput.audio`を使用
2. **サンプルレートの不一致**: 入力は16kHz、出力は24kHz
3. **mimeTypeのrate指定漏れ**: `audio/pcm;rate=16000`と明示的に指定
4. **VADによる応答トリガー失敗**: 十分な無音がない場合、手動VADを検討

## コミュニティ事例

### Google AI Developers Forum

- [エフェメラルトークンの問題](https://discuss.ai.google.dev/t/gemini-api-ephemeral-token-not-working/99122): v1betaではなくv1alphaを使用する必要がある
- [Live APIエフェメラルトークン問題](https://discuss.ai.google.dev/t/web-javascript-live-api-ephemeral-tokens-issue-and-latest-version-issue/90479): SDKバグでダブルスラッシュがURLに含まれる問題

## 結論・推奨

### 問題の原因（優先度順）

1. **メッセージ形式が誤り**: `{ media: {...} }`ではなく`{ realtimeInput: { audio: {...} } }`を使用すべき
2. **サンプルレートが24kHzで送信**: Geminiは16kHzを期待している
3. **mimeTypeにrateを指定していない**: `audio/pcm;rate=16000`と明示的に指定すべき

### 推奨される修正

```javascript
// useGeminiLive.ts の修正箇所

// 1. サンプルレートを16kHzに変更
const INPUT_SAMPLE_RATE = 16000;  // 入力用
const OUTPUT_SAMPLE_RATE = 24000; // 出力用

// 2. AudioContext のサンプルレート
const inputAudioContext = new AudioContext({ sampleRate: INPUT_SAMPLE_RATE });

// 3. メッセージ形式を修正
const audioMessage = {
  realtimeInput: {
    audio: {
      data: base64Data,
      mimeType: "audio/pcm;rate=16000"
    }
  }
};
ws.send(JSON.stringify(audioMessage));
```

### VAD調整（必要に応じて）

応答がトリガーされない場合、setup時にVAD感度を調整:

```javascript
realtimeInputConfig: {
  automaticActivityDetection: {
    disabled: false,
    endOfSpeechSensitivity: "END_SENSITIVITY_HIGH",  // より敏感に
    silenceDurationMs: 500  // 無音判定を500msに延長
  }
}
```

---

## 実装結果（2026-01-26 追記）

### 動作確認済み構成

以下の構成で Gemini Live API の音声会話が正常に動作することを確認:

| 項目 | 値 |
|------|------|
| モデル | `models/gemini-2.5-flash-native-audio-preview-12-2025` |
| WebSocket エンドポイント | `v1alpha` + `BidiGenerateContentConstrained` |
| 入力サンプルレート | 16 kHz |
| 出力サンプルレート | 24 kHz |
| メッセージ形式 | `{ realtimeInput: { audio: { data, mimeType } } }` |
| 動作確認ブラウザ | Safari（Chrome は AudioContext の問題あり） |

### 最終的な実装コード

```typescript
// 定数
const GEMINI_LIVE_WS_URL =
  "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContentConstrained";
const DEFAULT_MODEL = "models/gemini-2.5-flash-native-audio-preview-12-2025";
const INPUT_SAMPLE_RATE = 16000;
const OUTPUT_SAMPLE_RATE = 24000;

// setup メッセージ
const setupMessage = {
  setup: {
    model,
    generationConfig: {
      responseModalities: ["AUDIO"],
    },
    realtimeInputConfig: {
      automaticActivityDetection: {
        disabled: false,
        startOfSpeechSensitivity: "START_SENSITIVITY_HIGH",
        endOfSpeechSensitivity: "END_SENSITIVITY_HIGH",
        silenceDurationMs: 500,
      },
    },
    systemInstruction: {
      parts: [{ text: "You are a helpful voice assistant." }],
    },
  },
};

// 音声送信メッセージ
const audioMessage = {
  realtimeInput: {
    audio: {
      data: base64Data,
      mimeType: "audio/pcm;rate=16000",
    },
  },
};
```

### 判明した事実

1. **モデルの問題ではなかった**: `gemini-2.5-flash-native-audio-preview-12-2025` は正常に動作する。GitHub Issues で報告されていた「応答しない問題」は、メッセージ形式やサンプルレートの誤りが原因だった可能性が高い。

2. **VAD 設定の重要性**: `endOfSpeechSensitivity: "END_SENSITIVITY_HIGH"` と `silenceDurationMs: 500` を設定することで、発話終了の検出が安定する。

3. **Chrome の問題**: Chrome では `AudioContext({ sampleRate: 16000 })` を指定しても 24000Hz になる場合がある。Safari では正常に 16000Hz で動作。

### 残課題

- Chrome での音声入力問題の調査
- デバッグログの削除（本番環境向け）
- エラーハンドリングの強化

---

## ソース一覧

- [Live API - WebSockets API reference](https://ai.google.dev/api/live) - 公式APIリファレンス
- [Live API capabilities guide](https://ai.google.dev/gemini-api/docs/live-guide) - 公式ガイド
- [Get started with Live API](https://ai.google.dev/gemini-api/docs/live) - 公式入門ドキュメント
- [Ephemeral tokens](https://ai.google.dev/gemini-api/docs/ephemeral-tokens) - エフェメラルトークンドキュメント
- [Gemini Live API reference (Vertex AI)](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/model-reference/multimodal-live) - Vertex AI リファレンス
- [GitHub Issue #408 - No response for certain audios](https://github.com/google-gemini/cookbook/issues/408) - VAD問題
- [GitHub Issue #821 - Ephemeral token not working](https://github.com/google-gemini/cookbook/issues/821) - トークン問題
- [google-gemini/live-api-web-console](https://github.com/google-gemini/live-api-web-console) - 公式サンプルリポジトリ
- [Gemini 2.0 Realtime WebSocket API Notes](https://gist.github.com/quartzjer/9636066e96b4f904162df706210770e4) - コミュニティノート

## 関連資料

- このレポートを参照: /discuss, /plan で活用
- 実装修正の際は `/frontend-plan` で計画を作成することを推奨
