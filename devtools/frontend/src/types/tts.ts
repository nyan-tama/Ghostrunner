// TTS リクエストペイロード型。
// MVP++ では text のみ受け取り、speaker_id は backend env で固定。
export interface TTSRequest {
  text: string;
}
