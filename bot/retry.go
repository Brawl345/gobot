package bot

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

const (
	retryInitialDelay = time.Second
	retryMaxDelay     = time.Minute
)

func isPermanent(err error) bool {
	var tgErr *gotgbot.TelegramError
	if !errors.As(err, &tgErr) {
		return false
	}
	return tgErr.Code != http.StatusTooManyRequests && tgErr.Code < http.StatusInternalServerError
}

func retry[T any](ctx context.Context, method string, fn func(context.Context) (T, error)) (T, error) {
	delay := retryInitialDelay
	for attempt := 1; ; attempt++ {
		res, err := fn(ctx)
		if err == nil || isPermanent(err) {
			return res, err
		}

		log.Warn().
			Err(err).
			Str("method", method).
			Int("attempt", attempt).
			Dur("wait", delay).
			Msg("Telegram request failed, retrying")

		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		case <-time.After(delay):
		}
		delay = min(delay*2, retryMaxDelay)
	}
}
