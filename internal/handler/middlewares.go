package handler

import (
	"context"

	tg "gopkg.in/telebot.v3"
)

func (h *Handler) SetContext(ctx context.Context) func(next tg.HandlerFunc) tg.HandlerFunc {
	return func(next tg.HandlerFunc) tg.HandlerFunc {
		return func(c tg.Context) error {
			c.Set("ctx", ctx)
			return next(c)
		}
	}
}

func (h *Handler) CheckUser(next tg.HandlerFunc) tg.HandlerFunc {
	return func(c tg.Context) error {
		// если /start - идем регистрировать
		if c.Message().Text == "/start" {
			return next(c)
		}

		ctx := c.Get("ctx").(context.Context)
		userID := c.Sender().ID
		exists, err := h.Service.CheckUser(ctx, userID)
		if err != nil {
			return c.Send(err.Error())
		}

		if !exists {
			return c.Send("Выполните регистрацию командой: /start")
		}

		return next(c)
	}
}
