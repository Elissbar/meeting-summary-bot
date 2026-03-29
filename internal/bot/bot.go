package bot

import (
	"fmt"
	"io"
	"time"

	"github.com/Elissbar/meeting-summary-bot/internal/models"
	tg "gopkg.in/telebot.v3"
)

type Bot struct {
	Bot *tg.Bot
}

func NewBot(token string) (*Bot, error) {
	bot, err := tg.NewBot(
		tg.Settings{
			Token:  token,
			Poller: &tg.LongPoller{Timeout: 10 * time.Second},
		},
	)
	if err != nil {
		return nil, err
	}
	b := &Bot{
		Bot: bot,
	}

	return b, nil
}

func (b *Bot) GetFile(fileID, token string) (io.ReadCloser, error) {
	fileObj, err := b.Bot.FileByID(fileID)
	if err != nil {
		return nil, fmt.Errorf("error get fileObj: %v", err)
	}

	file, err := b.Bot.File(&fileObj)
	if err != nil {
		return nil, fmt.Errorf("error get file from TG: %v", err)
	}
	return file, nil
}

func (b *Bot) SendProcessedTasks(tasks <-chan models.Meeting) error {
	for task := range tasks {
		recipient := &tg.Chat{ID: task.UserID}
		var message string
		switch task.Status { // Всего может быть 4 статуса: "CANCELED", "DONE", "ERROR", "FAILED"
		case "DONE":
			message = fmt.Sprintf("Задача №%d обработана. Краткое описание: %s", task.ID, task.Summary)
		case "CANCELED":
			message = fmt.Sprintf("Задача №%d была отменена.", task.ID)
		default:
			message = fmt.Sprintf("Задача №%d была завершена с ошибкой.", task.ID)
		}
		b.Bot.Send(recipient, message)
	}
	return nil
}
