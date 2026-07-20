#!/bin/bash
# 質問待ち検知POC: Claudeが入力待ち(idle_prompt)になったらマーカーを書く。
# LLMは呼ばない(高速・毎ターン安全)。rawTail(最後の発言)は要約の材料として抽出のみ。
set -u
INPUT=$(cat)
MARKER_DIR="$HOME/.claude/gr-idle-markers"
mkdir -p "$MARKER_DIR"

CWD=$(printf '%s' "$INPUT" | jq -r '.cwd // ""')
SID=$(printf '%s' "$INPUT" | jq -r '.session_id // ""')
TP=$(printf '%s' "$INPUT" | jq -r '.transcript_path // ""')
EV=$(printf '%s' "$INPUT" | jq -r '.hook_event_name // ""')
NOW=$(date +%s)

# 観測用: 発火のたびに追記(本当にidle_promptが飛ぶかの証拠)
printf '%s\tevent=%s\tsid=%s\tcwd=%s\ttp=%s\n' "$NOW" "$EV" "$SID" "$CWD" "$TP" >> "$MARKER_DIR/_debug.log"

# rawTail抽出(best-effort)。行単位で fromjson? し壊れ行(Stop時に書き込み途中の最終行含む)をskip。
extract_last_assistant() {
  jq -sRr '[ split("\n")[] | fromjson? // empty | select(.type=="assistant") | .message.content[]? | select(.type=="text") | .text ] | last // ""' "$1" 2>/dev/null
}
extract_last_prompt() {
  jq -sRr '[ split("\n")[] | fromjson? // empty | select(.type=="last-prompt") | .lastPrompt ] | last // ""' "$1" 2>/dev/null
}
LAST_ASSISTANT=""
LAST_PROMPT=""
if [ -n "$TP" ] && [ -f "$TP" ]; then
  # Stop発火時は最終assistant行がまだ書き込み中のことがある。空なら短時間リトライ。
  for _try in 1 2 3; do
    LAST_ASSISTANT=$(extract_last_assistant "$TP")
    [ -n "$LAST_ASSISTANT" ] && break
    sleep 0.4
  done
  LAST_PROMPT=$(extract_last_prompt "$TP")
fi

# マーカー書き込み(先頭400字に切るのはjq内で。UTF-8境界を壊さない)
jq -n \
  --arg cwd "$CWD" --arg sid "$SID" --arg tp "$TP" --argjson ts "$NOW" \
  --arg la "$LAST_ASSISTANT" --arg lp "$LAST_PROMPT" \
  '{cwd:$cwd, session_id:$sid, transcript_path:$tp, timestamp:$ts,
    rawTail:{lastAssistant:($la[0:400]), lastPrompt:($lp[0:400])},
    summary:"", summarizedAt:""}' \
  > "$MARKER_DIR/${SID}.idle" 2>/dev/null

exit 0
