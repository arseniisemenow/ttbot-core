package handlers

import (
	"context"

	"github.com/arseniisemenow/ttbot-core/pkg/messenger"
)

// handleStart prints the full help text. Same handler covers /start and /help.
//
// Order: Matches → DM → Group config so the most-common-use commands sit
// at the top.
func (h *Handlers) handleStart(ctx context.Context, m *messenger.Message) error {
	const help = `ttbot — table-tennis match tracker.

Matches topic:
  /match — open an interactive opponent + score picker (recommended; no args needed)
  /match @opponent 3-1 — you vs opponent (your score first)
  /match @p1 @p2 3-1 — register a match between two named players
  /undo #N — undo or restore match #N (two-step confirm)
  /ping — bot reacts 👍 to your message and remembers your @username so others can /match @you

For the typed forms, each player token can be either @telegram_username or a bare S21 nickname.
Live leaderboards and per-player stats are auto-maintained as pinned messages in the stats topic.

DM:
  /start, /help — this message
  /login — store S21 creds so ttbot can call the identity service. Two-step
           (I prompt for creds in a reply and delete the reply immediately).
           Multiple users can /login; ttbot picks healthy stored credentials per call.
  /logout — remove your stored creds. Two-step confirm.
  /whoami — show whether you're logged in, the health of your creds, and
            how long until I auto-log-out if S21 stays unhappy.

Any topic of a registered group:
  /bot_register_group — link this group to ttbot
  /set_matches_topic — call inside the topic you want to use as the matches topic
  /set_stats_topic — call inside the topic you want to use as the stats topic
  /refresh_usernames — refresh the participants cache against Telegram`
	return h.reply(ctx, m, help)
}
