package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/Elissbar/meeting-summary-bot/internal/models"
	tg "gopkg.in/telebot.v3"
)

type Bot struct {
	Bot *tg.Bot
}

func NewBot(bot *tg.Bot) (*Bot, error) {
	b := &Bot{
		Bot: bot,
	}

	return b, nil
}

func (b *Bot) GetFile(fileID string) (io.Reader, string, error) {
	fileObj, err := b.Bot.FileByID(fileID)
	if err != nil {
		return nil, "", fmt.Errorf("error get fileObj: %v", err)
	}

	file, err := b.Bot.File(&fileObj)
	if err != nil {
		return nil, "", fmt.Errorf("error get file from TG: %v", err)
	}

	// Читаем первые 512 байт для определения MIME
	bufMime := make([]byte, 512)
	n, err := file.Read(bufMime)
	if err != nil {
		return nil, "", fmt.Errorf("error read file data: %v", err)
	}

	fileContent := io.MultiReader(bytes.NewReader(bufMime[:n]), file)

	// Файл может быть меньше 512 байт, передаем то кол-во, которое было прочитано, чтобы не передать пустые байты
	contentType := http.DetectContentType(bufMime[:n])
	var mime string
	switch t := contentType; t {
	case "audio/mpeg":
		mime = "MP3"
	case "audio/ogg", "application/ogg":
		mime = "OPUS"
	default:
		fmt.Printf("unsupported MIME type: %s", contentType)
		return nil, "", fmt.Errorf("unsupported MIME type: %s", contentType)
	}

	return fileContent, mime, nil
}

func (b *Bot) SendProcessedTasks(ctx context.Context, tasks <-chan models.Meeting) error {
	for task := range tasks {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		
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
