#!/usr/bin/env bash
# Poll for Telegram messages and transcribe any voice messages with Parakeet ASR.
# Usage: poll_with_transcribe.sh [--no-transcribe]
#
# Outputs JSON to stdout (same format as tg-bot poll), with voice message
# content replaced by transcription when voice_path is present.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CHAT_ID="${TGBOT_OWNER_ID:-}"
NO_TRANSCRIBE="${1:-}"

# Poll for messages
msgs=$(cd "$SCRIPT_DIR" && TELEGRAM_BOT_TOKEN="$TELEGRAM_BOT_TOKEN" \
    ./tg-bot poll ${CHAT_ID:+--chat "$CHAT_ID"} 2>/dev/null)

# Nothing to do
if [ "$msgs" = "null" ] || [ -z "$msgs" ]; then
    exit 0
fi

# If no transcription requested, just output raw
if [ "$NO_TRANSCRIBE" = "--no-transcribe" ]; then
    echo "$msgs"
    exit 0
fi

# Check if any message has a voice_path and transcribe it
echo "$msgs" | python3 -c "
import json, subprocess, sys

msgs = json.load(sys.stdin)
for msg in msgs:
    vp = msg.get('voice_path', '')
    if vp and msg.get('content', '') in ('[voice message]', ''):
        try:
            result = subprocess.run(
                ['uv', 'run', '$SCRIPT_DIR/transcribe.py', vp],
                capture_output=True, text=True, timeout=120
            )
            if result.returncode == 0 and result.stdout.strip():
                msg['content'] = result.stdout.strip()
            else:
                msg['content'] = '[voice message - transcription failed]'
        except Exception as e:
            msg['content'] = f'[voice message - error: {e}]'
json.dump(msgs, sys.stdout)
"
