package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Elissbar/meeting-summary-bot/internal/service"
	tg "gopkg.in/telebot.v3"
)

type Handler struct {
	Service *service.Service
	Bot     *tg.Bot
	log     *slog.Logger
}

func NewHandler(srvc *service.Service, tgBot *tg.Bot, log *slog.Logger) (*Handler, error) {
	return &Handler{srvc, tgBot, log}, nil
}

func (h *Handler) Handle(ctx context.Context) {
	h.Bot.Use(h.SetContext(ctx))
	h.Bot.Use(h.CheckUser)
	h.Bot.Handle(tg.OnVoice, h.ProcessMeeting)
	h.Bot.Handle(tg.OnAudio, h.ProcessMeeting)
	h.Bot.Handle("/start", h.CreateUser)
	h.Bot.Handle("/list", h.ListMeetings)
	h.Bot.Handle("/get", h.GetMeeting)
	h.Bot.Handle("/find", h.FindTranscription) // TODO: Стоит добавить поиск по нескольким ключевым словам.
	h.Bot.Handle("/chat", h.SendRequest)
}

func (h *Handler) CreateUser(c tg.Context) error {
	ctx := c.Get("ctx").(context.Context)

	userID := c.Sender().ID
	err := h.Service.CreateUser(ctx, userID)
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(fmt.Sprintf("User ID: %d", userID))
}

func (h *Handler) ListMeetings(c tg.Context) error {
	ctx := c.Get("ctx").(context.Context)

	userID := c.Sender().ID
	h.log.Info("", "UserID:", userID)

	meetings, err := h.Service.GetAllMeetings(ctx, userID)
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(fmt.Sprintf("Список ваших сохраненных встреч: %s", strings.Join(meetings, ", ")))
}

func (h *Handler) GetMeeting(c tg.Context) error {
	ctx := c.Get("ctx").(context.Context)

	userID := c.Sender().ID
	h.log.Info("", "UserID:", userID)

	args := c.Args()

	if len(args) == 0 {
		return c.Send("Пожалуйста, укажите ID записи. Пример: /get 123")
	}

	chCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	transcript, err := h.Service.GetMeeting(chCtx, args[0])
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(fmt.Sprintf("Текст встречи: \n%s", transcript))
}

func (h *Handler) FindTranscription(c tg.Context) error {
	ctx := c.Get("ctx").(context.Context)

	userID := c.Sender().ID
	h.log.Info("", "UserID:", userID)

	args := c.Args()

	if len(args) == 0 {
		return c.Send("Пожалуйста, укажите ключевое слово для поиска. Пример: /find keyword")
	}

	chCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	transcript, err := h.Service.FindTranscription(chCtx, args[0])
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(fmt.Sprintf("Текст встречи: \n%s", transcript))
}

func (b *Handler) ProcessMeeting(c tg.Context) error {
	ctx := c.Get("ctx").(context.Context)
	userID := c.Sender().ID

	var fileID string
	if c.Message().Audio == nil && c.Message().Voice != nil {
		b.log.Info("Голосовое от пользователя", "UserID:", userID)
		fileID = c.Message().Voice.FileID
	} else if c.Message().Audio != nil && c.Message().Voice == nil {
		b.log.Info("Аудио от пользователя", "UserID:", userID)
		fileID = c.Message().Audio.FileID
	} else {
		c.Send(fmt.Sprintf("Unexpected message type. Voice: %v. Audio: %v", c.Message().Voice, c.Message().Audio))
		return fmt.Errorf("unexpected message type.")
	}

	taskID, err := b.Service.CreateTask(ctx, fileID, userID)
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(fmt.Sprintf("File in process. ID: %d", taskID))
}

func (b *Handler) SendRequest(c tg.Context) error {
	prompt := c.Message().Payload

	response, err := b.Service.RequestToChat(prompt)
	if err != nil {
		return c.Send(err.Error())
	}

	return c.Send(response)
}
