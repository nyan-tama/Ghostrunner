// TTS リクエストペイロード型。
// MVP+ では text のみ受け取り、voice_id / model_id は backend env で固定。
// 将来 Voice 選択 UI (MVP++) を導入する際は voiceId / modelId を任意フィールドとして拡張する。
export interface TTSRequest {
  text: string;
}
