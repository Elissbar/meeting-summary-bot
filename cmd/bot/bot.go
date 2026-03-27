package main

import (
	"fmt"

	"github.com/Elissbar/meeting-summary-bot/internal/config"
	"github.com/Elissbar/meeting-summary-bot/internal/gigachat"
	"github.com/Elissbar/meeting-summary-bot/internal/handler"
	"github.com/Elissbar/meeting-summary-bot/internal/salutespeech"
	"github.com/Elissbar/meeting-summary-bot/internal/service"
)

func main() {
	config, err := config.NewConfig()
	if err != nil {
		panic(fmt.Errorf("get config: %w", err))
	}

	salute := salutespeech.NewSaluteSpeechClient(config.AuthURL, config.SaluteAuthToken, config.SaluteScope)
	giga := gigachat.NewGigaChatClient(config.AuthURL, config.GigaChatAuthToken, config.GigaChatScope)
	
	serv := service.NewService(salute, giga)

	tgBot, err := handler.NewBot(config.BotToken, serv)
	if err != nil {
		panic(fmt.Errorf("create bot error: %w", err))
	}
	tgBot.Handle()

}
