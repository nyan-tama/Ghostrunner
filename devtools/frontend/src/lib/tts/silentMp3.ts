// iOS Safari 等の autoplay unlock に使う、極短 (約 0.1 秒) の無音 MP3 を data URL として保持する。
//
// 用途:
//   useTTS.prime() がユーザージェスチャの同期スコープで
//   `<audio>` に `src = SILENT_MP3_DATA_URL` をセット → `play()` → `pause()` を行うことで、
//   以降の非同期 `audio.play()` が UA に握りつぶされないようにパイプを開く。
//
// 生成元コマンド (参考):
//   ffmpeg -f lavfi -i anullsrc=r=44100:cl=mono -t 0.1 -c:a libmp3lame -b:a 128k silent.mp3
//   base64 -i silent.mp3
//
// 下記の値は ID3v2 ヘッダ + 極短の無音 MP3 フレームを含む base64 表現で、
// `<audio>` の src として再生・即停止可能な最小実装。
// MP3 デコーダの細かい挙動差で再生時間がゼロ秒寄りになることがあるが、
// unlock 用途では「ロード→play→即 pause」が完了すれば足りる。
export const SILENT_MP3_DATA_URL =
  "data:audio/mpeg;base64,SUQzBAAAAAAAI1RTU0UAAAAPAAADTGF2ZjU5LjI3LjEwMAAAAAAAAAAAAAAA//tQwAAAAAAAAAAAAAAAAAAAAAAASW5mbwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA";
