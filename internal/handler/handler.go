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
	// Подумать над каналом для сохранения батчей задач. 
	// Накапливать запросы от пользователей и при наполнении сохранять весь батч.
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
}

func (b *TGBot) Start() {
	b.Bot.Start()
}

func (b *TGBot) Stop() {
	b.Bot.Stop()
}

func (b *TGBot) OnVoice(c tg.Context) error {
	voice := c.Message().Voice
	userID := c.Sender().ID
	fmt.Println("Voice:", voice)

	rc, err := b.Bot.File(&voice.File)
	if err != nil {
		return c.Send(fmt.Sprintf("Could not get file info.\nError: %s", err.Error()))
	}
	defer rc.Close()

	taskID, err := b.Service.UploadAndStartProcess(rc, userID)
	if err != nil {
		return c.Send(fmt.Sprintf("Error: %s", err.Error()))
	}

	return c.Send(fmt.Sprintf("File in process. ID: %d", taskID))
}
