package handlers

import (
	"context"
	"errors"
	"regexp"
	"strings"

	s21account "github.com/arseniisemenow/s21-account-go"

	"github.com/arseniisemenow/ttbot-core/pkg/messenger"
)

// logoutPromptRegex matches the /logout confirmation prompt state tag.
// Same shape as the login regex — tag sits at the end inside a spoiler.
var logoutPromptRegex = regexp.MustCompile(`\[LOGIN_OP=logout\]`)

// handleLogout starts the two-step /logout flow. Caller must be logged in.
func (h *Handlers) handleLogout(ctx context.Context, m *messenger.Message) error {
	if _, err := h.Store.S21Accounts().Get(ctx, m.From.ID); errors.Is(err, s21account.ErrNotFound) {
		return h.reply(ctx, m, "You're not logged in — nothing to log out from.")
	}
	prompt := "You are about to log out (your stored S21 credentials for ttbot will be deleted).\n\n" +
		"After this:\n" +
		"- Other logged-in users continue to back ttbot's S21 calls; only your stored credentials are removed.\n" +
		"- Group registration, matches, rankings — all keep working as long as at least one healthy login remains.\n\n" +
		"Reply with `confirm` to proceed. Any other reply cancels." +
		"\n\n" + spoilerWrap("[LOGIN_OP=logout]")
	if _, err := h.M.SendMessageWithForceReplyHTML(ctx, m.Chat.ID, prompt, "confirm"); err != nil {
		return h.userFacingError(ctx, m, "/logout: send prompt",
			"Telegram is unreachable right now — try /logout again shortly.", err)
	}
	return nil
}

// isLogoutReply detects the confirm reply.
func isLogoutReply(m *messenger.Message) bool {
	if m == nil || m.ReplyTo == nil || m.ReplyTo.From == nil || !m.ReplyTo.From.IsBot {
		return false
	}
	return logoutPromptRegex.MatchString(m.ReplyTo.Text)
}

// handleLogoutReply parses the confirm reply. Anything other than "confirm"
// cancels.
func (h *Handlers) handleLogoutReply(ctx context.Context, m *messenger.Message) error {
	if strings.TrimSpace(strings.ToLower(m.Text)) != "confirm" {
		return h.reply(ctx, m, "Cancelled — you are still logged in.")
	}
	if err := h.Store.S21Accounts().Delete(ctx, m.From.ID); err != nil {
		return h.userFacingError(ctx, m, "/logout: delete account",
			"The database is unreachable right now — try /logout again shortly.", err)
	}
	return h.reply(ctx, m, "Logged out. Your stored S21 credentials have been removed. /login again whenever you want.")
}
