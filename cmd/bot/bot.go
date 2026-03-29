package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Elissbar/meeting-summary-bot/internal/bot"
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
	grp, gCtx := errgroup.WithContext(shutdownCtx)
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

	tgBot, err := bot.NewBot(config.BotToken)
	if err != nil {
		panic(fmt.Errorf("create Bot error: %w", err))
	}

	var wg sync.WaitGroup
	serv := service.NewService(gCtx, salute, giga, storage, config, &wg, tgBot)

	handler, err := handler.NewHandler(serv, tgBot.Bot)
	if err != nil {
		panic(fmt.Errorf("create bot error: %w", err))
	}

	grp.Go(func() error {
		handler.Handle() // Как обработать ошибку
		tgBot.Bot.Start()
		return nil
	})
	grp.Go(func() error {
		<-gCtx.Done()
		tgBot.Bot.Stop()
		close(serv.Tasks)
		close(serv.Results)
		close(serv.GigaTasks)
		return nil
	})

	if err := grp.Wait(); err != nil {
		fmt.Printf("Application error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Bot stopped")
}
