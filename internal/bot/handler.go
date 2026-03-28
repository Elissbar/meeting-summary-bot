package bot

import (
	"fmt"

	"github.com/Elissbar/meeting-summary-bot/internal/service"
	tg "gopkg.in/telebot.v3"
)

type Handler struct {
	Service *service.Service
}

func NewHandler(srvc *service.Service) (*Handler, error) {
	return &Handler{srvc}, nil
}

func (h *Handler) Handle() {
	h.Service.Bot.Handle(tg.OnVoice, h.OnVoice)
}

func (b *Handler) OnVoice(c tg.Context) error {
	fileID := c.Message().Voice.FileID
	userID := c.Sender().ID
	fmt.Printf("FileID: %s. UserID: %d.\n", fileID, userID)

	taskID, err := b.Service.CreateTask(fileID, userID)
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(fmt.Sprintf("File in process. ID: %d", taskID))
}
