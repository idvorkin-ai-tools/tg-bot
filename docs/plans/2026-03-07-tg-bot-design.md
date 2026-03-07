# tg-bot: Telegram Bridge for Claude Code

## Problem

Run Claude Code as the AI brain behind a Telegram bot, without nanoclaw's full stack. Need a fast, minimal bridge between Telegram and Claude Code.

## Decision

Single Go binary with two modes: a long-running listener that receives Telegram messages into SQLite, and instant CLI subcommands that Claude Code calls to poll messages and send replies.

## Architecture

```
Telegram API <---> tg-bot listen (long-running) ---> SQLite
                                                       ^
                                                       |
                   Claude Code hooks/bash calls:       |
                   tg-bot poll / send / chats ----------+
```

## Tech Stack

- **Language**: Go
- **Telegram library**: telebot v4 (gopkg.in/telebot.v4)
- **Storage**: SQLite via modernc.org/sqlite (pure Go, no CGO)
- **Config**: `TELEGRAM_BOT_TOKEN` env var
- **DB location**: `~/.local/share/tg-bot/tg-bot.db` (XDG)

## CLI Subcommands

| Command | Description | Output |
|---------|-------------|--------|
| `tg-bot listen` | Long-running Telegram poller, writes to SQLite | Logs to stderr |
| `tg-bot poll` | Return unread messages, mark as read | JSON array |
| `tg-bot poll --chat <id>` | Filter to specific chat | JSON array |
| `tg-bot send <chat_id> <message>` | Send message to chat | Sent message JSON |
| `tg-bot send --topic <id> <chat_id> <message>` | Send to a forum topic | Sent message JSON |
| `tg-bot chats` | List known chats/groups | JSON array |
| `tg-bot typing <chat_id>` | Send typing indicator | (silent) |

## SQLite Schema

```sql
CREATE TABLE messages (
    id INTEGER PRIMARY KEY,
    telegram_msg_id INTEGER,
    chat_id INTEGER,
    topic_id INTEGER,
    sender_name TEXT,
    sender_id INTEGER,
    content TEXT,
    timestamp TEXT,
    read INTEGER DEFAULT 0
);

CREATE TABLE chats (
    chat_id INTEGER PRIMARY KEY,
    chat_type TEXT,
    title TEXT,
    is_forum INTEGER DEFAULT 0,
    last_seen TEXT
);
```

## Claude Code Integration

Claude Code calls the CLI directly via bash:

```bash
# Poll for new messages
tg-bot poll

# Send a reply
tg-bot send 123456789 "Hello from Claude!"

# Reply in a forum topic
tg-bot send --topic 42 123456789 "Threaded reply"
```

Can also wire into hooks:

```json
{
  "hooks": {
    "prompt-submit": [{
      "command": "tg-bot poll"
    }]
  }
}
```

## Inspiration

Channel architecture inspired by [nanoclaw](https://github.com/qwibitai/nanoclaw), which uses grammy (JS) for Telegram with a similar message queue pattern. This is a stripped-down Go reimplementation of just the transport layer.
