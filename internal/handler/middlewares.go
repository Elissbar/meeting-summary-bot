package handler

import (
	tg "gopkg.in/telebot.v3"
)

func (h *Handler) CheckUser(next tg.HandlerFunc) tg.HandlerFunc {
	return func(c tg.Context) error {
		// если /start - идем регистрировать
		if c.Message().Text == "/start" {
			return next(c)
		}
		
		userID := c.Sender().ID
		exists, err := h.Service.CheckUser(userID)
		if err != nil {
			return c.Send(err.Error())
		}
		
		if !exists {
			return c.Send("Выполните регистрацию командой: /start")
		}

		return next(c)
	}
}