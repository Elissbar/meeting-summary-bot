package bot

import (
	"time"

	tg "gopkg.in/telebot.v3"
)

func NewBot(token string) (*tg.Bot, error) {
	return tg.NewBot(
		tg.Settings{
			Token:  token,
			Poller: &tg.LongPoller{Timeout: 10 * time.Second},
		},
	)
}
