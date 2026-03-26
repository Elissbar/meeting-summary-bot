package handler

import (
	"fmt"
	"time"

	"github.com/Elissbar/meeting-summary-bot/internal/service"
	tg "gopkg.in/telebot.v3"
)

type TGBot struct {
	Bot     *tg.Bot
	Service *service.Service
}

func NewBot(token string, srvc *service.Service) (*TGBot, error) {
	bot, err := tg.NewBot(
		tg.Settings{
			Token:  token,
			Poller: &tg.LongPoller{Timeout: 10 * time.Second},
		},
	)
	if err != nil {
		return nil, err
	}

	return &TGBot{bot, srvc}, nil
}

func (b *TGBot) Handle() {
	b.Bot.Handle(tg.OnVoice, b.OnVoice)
	b.Bot.Start()
}

func (b *TGBot) OnVoice(c tg.Context) error {
	voice := c.Message().Voice
	fmt.Println("Voice:", voice)

	rc, err := b.Bot.File(&voice.File)
	if err != nil {
		return c.Send(fmt.Sprintf("Could not get file info.\nError: %s", err.Error()))
	}
	defer rc.Close()

	fileID, err := b.Service.UploadAndStartProcess(rc)
	if err != nil {
		return c.Send(fmt.Sprintf("Error: %s", err.Error()))
	}

	return c.Send("File in process: "+fileID)
}
