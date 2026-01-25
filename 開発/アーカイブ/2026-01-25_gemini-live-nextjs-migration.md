# 検討: Gemini Live API実装前にNext.js化すべきか

## 概要

Gemini Live API（Function Calling対応の音声AIインターフェース）を実装する前に、現在のバニラJS UI（`web/index.html`）をNext.jsに移行すべきかどうかの検討。

---

## 現状整理

### 既存のUIアーキテクチャ

```
現在:
web/index.html (バニラJS, 約1000行)
  - SSEストリーミング対応
  - 質問への回答UI
  - ファイルセレクター
  - イベント表示

計画中:
frontend/ (Next.js 15 + React 19)
  - まだ存在しない
  - 以前の計画ではOpenAI Realtime API + Whisper想定
```

### Gemini Live APIの特徴

| 項目 | 内容 |
|------|------|
| 通信方式 | WebSocket |
| 音声入力 | 16-bit PCM, 16kHz（ダウンサンプリング必要） |
| 音声出力 | 16-bit PCM, 24kHz |
| 認証 | エフェメラルトークン推奨 |
| Function Calling | サポート（UI操作 + Claude CLI） |
| セッション時間 | 最大15分 |

### 実装の複雑さ

Gemini Live APIの実装には以下が必要:

1. **WebSocket接続管理**: 接続、再接続、セッション再開
2. **Web Audio API**: マイク入力のダウンサンプリング、音声出力の再生
3. **状態管理**: 接続状態、発話状態、ツール呼び出し状態
4. **Function Call実行**: UI操作とClaude CLI実行の連携
5. **エフェメラルトークン取得**: バックエンドAPI必要

---

## 選択肢の比較

### A案: 先にNext.js化してからGemini Live実装

**メリット:**
- React/TypeScriptの型安全性でオーディオ処理やWebSocket管理のバグを減らせる
- カスタムフックで状態管理を整理できる（`useGeminiLive`, `useAudioProcessor`）
- コンポーネント分割で責務が明確になる
- 既存の`web/index.html`の機能を整理しながら移植できる
- 将来的な保守性が高い

**デメリット:**
- Next.js移行自体に時間がかかる
- 既存機能の再実装が必要（SSEストリーミング、質問UI等）
- Gemini Live実装開始までの遅延

**推定工数:**
- Next.js基盤構築: 中程度
- 既存機能移植: 中程度
- Gemini Live実装: 中程度

### B案: 現在のバニラJSにGemini Live APIを追加

**メリット:**
- すぐにGemini Live実装を開始できる
- 既存UIは動作確認済み

**デメリット:**
- `web/index.html`が2000行以上に膨張する可能性
- 状態管理が複雑化（グローバル変数の増加）
- Web Audio API + WebSocketのコードがスパゲッティ化しやすい
- TypeScriptの恩恵を受けられない（オーディオ処理のバグが発見しにくい）
- 後からNext.jsに移行する際、全部作り直しになる

**推定工数:**
- Gemini Live実装: 中程度（ただしバグ修正に時間がかかる可能性）
- 後でNext.js移行: 大規模なリファクタリング必要

### C案: 並行開発（既存UIを維持しつつNext.jsで新機能）

**メリット:**
- 既存機能は継続利用可能
- Gemini Live APIはNext.jsでクリーンに実装
- 段階的な移行が可能

**デメリット:**
- 2つのUIをメンテナンスする負担
- 重複コード（API呼び出し等）

**推定工数:**
- Next.js基盤構築: 中程度
- Gemini Live実装: 中程度
- 既存機能は後で移植

---

## 分析

### 技術的な観点

Gemini Live APIの実装では以下の複雑な処理が必要:

1. **音声処理**
   - マイク入力（44.1kHz/48kHz）を16kHzにダウンサンプリング
   - Float32 -> Int16 PCM変換
   - 24kHz PCMの再生

2. **WebSocket状態管理**
   - 接続状態（disconnected/connecting/connected/error）
   - セッション設定
   - セッション再開
   - エラーハンドリング

3. **対話状態管理**
   - ユーザー発話中/AI発話中
   - 割り込み（barge-in）処理
   - ペンディング中のツール呼び出し

**バニラJSの課題:**
- グローバル変数による状態管理はデバッグが困難
- 音声処理のバグは型がないと発見しにくい
- コンポーネント化されていないとUIの状態遷移が追いにくい

**React/TypeScriptの利点:**
- カスタムフックで複雑な状態を分離
- 型定義でオーディオ処理のミスを防止
- コンポーネントごとに責務を明確化

### 既存計画との整合性

以前の計画書（`2026-01-25_function-calling-ui_plan.md`）では:
- Next.js 15 + React 19を前提としていた
- `useSSEStream`, `useVoiceInput`, `useAIChat`等のフック設計済み
- 既存`web/index.html`は「両方維持」として移行期間の安全網に

→ Gemini Live APIに切り替えても、この構成は有効

---

## 推奨案

### C案（並行開発）を推奨

**理由:**

1. **既存機能への影響なし**: 現在動作している`web/index.html`はそのまま維持
2. **Gemini Live APIをクリーンに実装**: 最初からReact + TypeScriptで複雑な音声処理を管理
3. **段階的な移行が可能**: Gemini Live UIが安定したら、既存機能を徐々に移植
4. **以前の計画と整合**: 既存の計画書のアーキテクチャをそのまま活用できる

### 具体的な進め方

```
Phase 1: Next.js基盤構築
- frontend/ ディレクトリ作成
- 基本的なプロジェクト構成
- Go APIとの接続確認
- エフェメラルトークン発行API追加

Phase 2: Gemini Live API実装
- WebSocket接続（useGeminiLive）
- 音声処理（useAudioProcessor）
- Function Calling実行
- 基本的な音声UI

Phase 3: 機能統合
- 既存機能（/plan, /fullstack等）をNext.jsから呼び出し
- 質問UIの移植
- イベント表示の移植

Phase 4: 移行完了
- web/index.html 削除
- Next.jsをメインUIに
```

### 代替案: A案（先にNext.js化）

C案より保守的なアプローチとして、Gemini Live実装前に既存機能を完全にNext.jsに移植する選択肢もある。

**A案が適切な場合:**
- Gemini Live実装までに少し時間がある
- 既存UIの全機能を確実にNext.jsで動作させたい
- 2つのUIをメンテナンスしたくない

---

## 結論

**推奨: C案（並行開発）**

- 既存の`web/index.html`は維持
- Gemini Live APIはNext.jsで新規実装
- 段階的に機能を移植

**ただし、ユーザーの優先度によってはA案も有効**
- 「確実に動く基盤を先に作りたい」→ A案
- 「早くGemini Liveを試したい」→ C案

---

## 次のステップ

方針決定後:
1. `/plan` でNext.js基盤 + Gemini Live APIの実装計画を作成
2. Phase 1から段階的に実装
