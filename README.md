# tg-bot: Telegram Bridge for Claude Code

A minimal Go CLI that bridges Telegram to Claude Code via SQLite. The bot receives Telegram messages into a local database, and Claude Code polls for new messages and sends replies through simple CLI commands.

## Architecture

```
Telegram API ←→ tg-bot listen (daemon) ←→ SQLite ←→ tg-bot poll/send (CLI)
```

- **`listen`** — long-running daemon that receives messages from Telegram via long-polling
- **`poll`** — instant CLI command that returns unread messages and marks them read
- **`send`** — sends a reply to a chat
- **`typing`** — sends a typing indicator
- **`chats`** — lists known chats

## Setup

### 1. Create a Telegram bot

Talk to [@BotFather](https://t.me/BotFather) on Telegram and create a new bot. Save the token.

### 2. Build

```bash
go build -o tg-bot .
```

### 3. Environment variables

```bash
export TELEGRAM_BOT_TOKEN="your-token-here"
export TGBOT_OWNER_ID="your-telegram-user-id"  # optional: only accept messages from this user
```

To find your user ID, start the bot without `TGBOT_OWNER_ID` and send it `/chatid`.

### 4. Start the listener

```bash
# Run in foreground
./tg-bot listen

# Or in tmux for persistence
tmux new-session -d -s tgbot './tg-bot listen'
```

### 5. Test it

Send a message to your bot in Telegram, then:

```bash
./tg-bot poll     # see the message
./tg-bot send <chat_id> "Hello from the CLI!"
```

## Claude Code Integration

### Quick setup (manual polling)

Add to your Claude Code session:

```bash
# Poll and respond
./tg-bot poll
./tg-bot typing <chat_id>
./tg-bot send <chat_id> "your reply"
```

### Automated setup (cron + polling loop)

Run a fast polling loop that writes incoming messages to a file:

```bash
# polling-loop.sh
while true; do
  result=$(./tg-bot poll 2>/dev/null)
  if [ "$result" != "null" ] && [ -n "$result" ]; then
    echo "$result" >> /tmp/tg-bot-incoming.jsonl
  fi
  sleep 5
done
```

Start it in tmux:

```bash
tmux new-session -d -s tgpoll 'bash polling-loop.sh'
```

Then use a Claude Code cron to check the file and respond:

```
/loop 1m Check /tmp/tg-bot-incoming.jsonl for new messages and respond
```

### Full session setup (3 tmux sessions)

```bash
# 1. Listener — receives from Telegram API
tmux new-session -d -s tgbot \
  'TELEGRAM_BOT_TOKEN="..." TGBOT_OWNER_ID="..." ./tg-bot listen'

# 2. Poller — checks for new messages every 5s
tmux new-session -d -s tgpoll \
  'while true; do
    result=$(./tg-bot poll 2>/dev/null)
    if [ "$result" != "null" ] && [ -n "$result" ]; then
      echo "$result" >> /tmp/tg-bot-incoming.jsonl
    fi
    sleep 5
  done'

# 3. Claude Code — picks up messages and responds as Larry (or whatever persona)
# Use /loop or cron to check /tmp/tg-bot-incoming.jsonl
```

## CLI Reference

| Command | Description | Output |
|---------|-------------|--------|
| `tg-bot listen` | Long-running Telegram poller | Logs to stderr |
| `tg-bot poll [--chat ID]` | Return unread messages, mark read | JSON array |
| `tg-bot send <chat_id> <message> [--topic ID]` | Send message | `{"message_id", "chat_id"}` |
| `tg-bot typing <chat_id>` | Send typing indicator | (silent) |
| `tg-bot chats` | List known chats | JSON array |

## Configuration

| Env var | Required | Description |
|---------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | Yes | Bot token from @BotFather |
| `TGBOT_OWNER_ID` | No | Only accept messages from this Telegram user ID |
| `TGBOT_DB` | No | Override database path (default: `~/.local/share/tg-bot/tg-bot.db`) |

## Built-in bot commands

Send these to the bot in Telegram:

- `/chatid` — replies with the chat ID (useful for setup)
- `/ping` — replies with "pong" (health check)
