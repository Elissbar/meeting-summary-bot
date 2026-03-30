package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	h.Bot.Handle(tg.OnVoice, h.ProcessMeeting)
	h.Bot.Handle(tg.OnAudio, h.ProcessMeeting)
	h.Bot.Handle("/start", h.CreateUser)
	h.Bot.Handle("/list", h.ListMeetings)
	h.Bot.Handle("/get", h.GetMeeting)
}

func (h *Handler) CreateUser(c tg.Context) error {
	userID := c.Sender().ID
	err := h.Service.CreateUser(userID)
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(fmt.Sprintf("User ID: %d", userID))
}

func (h *Handler) ListMeetings(c tg.Context) error {
	userID := c.Sender().ID
	fmt.Printf("UserID: %d.\n", userID)

	ctx, cancel := context.WithTimeout(h.Service.Ctx, time.Second*5)
	defer cancel()

	meetings, err := h.Service.Storage.GetAllMeetings(ctx, userID)
	if err != nil {
		return c.Send(err.Error())
	}

	meetingsStr := make([]string, len(meetings))
	for i, v := range meetings {
		meetingsStr[i] = strconv.FormatInt(v, 10) // Convert int to string
	}
	return c.Send(fmt.Sprintf("Список ваших сохраненных встреч: %s", strings.Join(meetingsStr, ", ")))
}

func (h *Handler) GetMeeting(c tg.Context) error {
	userID := c.Sender().ID
	fmt.Printf("UserID: %d.\n", userID)

	args := c.Args()

	if len(args) == 0 {
		return c.Send("Пожалуйста, укажите ID записи. Пример: /get 123")
	}

	ctx, cancel := context.WithTimeout(h.Service.Ctx, time.Second*5)
	defer cancel()
	
	transcript, err := h.Service.Storage.GetMeeting(ctx, args[0])
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(fmt.Sprintf("Текст встречи: \n%s", transcript))
}

func (b *Handler) ProcessMeeting(c tg.Context) error {
	var fileID string
	if c.Message().Audio == nil && c.Message().Voice != nil {
		fmt.Println("Пришло голосовое")
		fileID = c.Message().Voice.FileID
	} else if c.Message().Audio != nil && c.Message().Voice == nil {
		fmt.Println("Пришло аудио")
		fileID = c.Message().Audio.FileID
	} else {
		c.Send(fmt.Sprintf("Unexpected message type. Voice: %v. Audio: %v", c.Message().Voice, c.Message().Audio))
		return fmt.Errorf("unexpected message type.")
	}

	userID := c.Sender().ID
	fmt.Printf("FileID: %s. UserID: %d.\n", fileID, userID)

	taskID, err := b.Service.CreateTask(fileID, userID)
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(fmt.Sprintf("File in process. ID: %d", taskID))
}
