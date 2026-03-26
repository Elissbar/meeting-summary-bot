package main

import (
	"time"

	"github.com/Elissbar/meeting-summary-bot/internal/handler"
	tg "gopkg.in/telebot.v3"
)

func main() {
	b, err := tg.NewBot(
		tg.Settings{
			Token: "7925416453:AAFa-3-AhEel_8wQm-VNlvJD8bnPlzRtQOs",
			Poller: &tg.LongPoller{Timeout: 10 * time.Second},
		},
	)
	if err != nil {
		return
	}

	b.Handle(tg.OnVoice, handler.OnVoice(b))

	b.Start()
}
