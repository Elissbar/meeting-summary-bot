package handler

import (
	"fmt"

	"github.com/Elissbar/meeting-summary-bot/internal/service"
	tg "gopkg.in/telebot.v3"
)

type Handler struct {
	Service *service.Service
	Bot     *tg.Bot
}

func NewHandler(srvc *service.Service, tgBot *tg.Bot) (*Handler, error) {
	return &Handler{srvc, tgBot}, nil
}

func (h *Handler) Handle() {
	h.Bot.Use(h.CheckUser)
	h.Bot.Handle(tg.OnVoice, h.OnVoice)
	h.Bot.Handle("/start", h.CreateUser)
}

func (h *Handler) CreateUser(c tg.Context) error {
	userID := c.Sender().ID
	err := h.Service.CreateUser(userID)
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(fmt.Sprintf("User ID: %d", userID))
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
