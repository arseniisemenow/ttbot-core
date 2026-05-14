package handlers

import (
	"context"
	"errors"
	"fmt"

	s21account "github.com/arseniisemenow/s21-account-go"

	"github.com/arseniisemenow/ttbot-core/pkg/messenger"
)

// handleWhoami renders the caller's S21 account state (login, campus,
// last-used, health), plus a count of how many other accounts are
// currently logged in. Returns "you're not logged in" if the caller
// has no stored credentials.
func (h *Handlers) handleWhoami(ctx context.Context, m *messenger.Message) error {
	a, err := h.Store.S21Accounts().Get(ctx, m.From.ID)
	if errors.Is(err, s21account.ErrNotFound) {
		return h.reply(ctx, m, "You're not logged in. Run /login to register your S21 credentials.")
	}
	if err != nil {
		return h.userFacingError(ctx, m, "/whoami: read account",
			"The database is unreachable right now — try again shortly.", err)
	}
	body := s21account.RenderWhoami(a, h.Config.Now())
	// Append the count of OTHER logged-in accounts. Helps a user gauge
	// "am I the only one keeping this bot alive?" Errors here are
	// silently ignored — the count is a nice-to-have, not load-bearing.
	if all, listErr := h.Store.S21Accounts().List(ctx); listErr == nil {
		others := 0
		for _, row := range all {
			if row.TelegramID != m.From.ID {
				others++
			}
		}
		body += "\n\n" + otherAccountsLine(others)
	}
	return h.reply(ctx, m, body)
}

// otherAccountsLine produces a small footer like "Together with 3 other
// logged-in account(s)." with grammar that's correct for 0, 1, and N.
func otherAccountsLine(n int) string {
	switch n {
	case 0:
		return "You're the only logged-in account right now."
	case 1:
		return "Together with 1 other logged-in account."
	default:
		return fmt.Sprintf("Together with %d other logged-in accounts.", n)
	}
}
