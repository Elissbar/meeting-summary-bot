package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/Elissbar/meeting-summary-bot/internal/config"
	"github.com/Elissbar/meeting-summary-bot/internal/gigachat"
	"github.com/Elissbar/meeting-summary-bot/internal/handler"
	"github.com/Elissbar/meeting-summary-bot/internal/salutespeech"
	"github.com/Elissbar/meeting-summary-bot/internal/service"
	"github.com/Elissbar/meeting-summary-bot/internal/storage"
	"golang.org/x/sync/errgroup"
)

func main() {
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	config, err := config.NewConfig()
	if err != nil {
		panic(fmt.Errorf("get config: %w", err))
	}

	salute := salutespeech.NewSaluteSpeechClient(config.AuthURL, config.SaluteAuthToken, config.SaluteScope)
	giga := gigachat.NewGigaChatClient(config.AuthURL, config.GigaChatAuthToken, config.GigaChatScope)

	storage, err := storage.NewDatabaseStorage(config.DBConnectionURI)
	if err != nil {
		panic(fmt.Errorf("create storage error: %w", err))
	}

	serv := service.NewService(salute, giga, storage, config.WaitPlaceInChan, config.StopProcess)

	tgBot, err := handler.NewBot(config.BotToken, serv)
	if err != nil {
		panic(fmt.Errorf("create bot error: %w", err))
	}

	grp, gCtx := errgroup.WithContext(shutdownCtx)
	grp.Go(func() error {
		tgBot.Handle() // Как обработать ошибку
		tgBot.Start()
		return nil
	})
	grp.Go(func() error {
		<-gCtx.Done()
		tgBot.Stop()
		close(serv.Tasks)
		return nil
	})
}
