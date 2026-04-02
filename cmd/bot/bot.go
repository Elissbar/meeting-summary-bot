package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	tg "gopkg.in/telebot.v3"

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
	storage, err := storage.NewStorage(config.DBConnectionURI)
	if err != nil {
		panic(fmt.Errorf("create storage error: %w", err))
	}

	tgBot, err := tg.NewBot(
		tg.Settings{Token:  config.BotToken, Poller: &tg.LongPoller{Timeout: 10*time.Second}},
	)
	if err != nil {
		panic(fmt.Errorf("create bot error: %w", err))
	}

	botClient, err := bot.NewBot(tgBot)
	if err != nil {
		panic(fmt.Errorf("create Bot error: %w", err))
	}

	var wg sync.WaitGroup
	serv := service.NewService(salute, giga, storage, config, &wg, botClient)

	handler, err := handler.NewHandler(serv, botClient.Bot)
	if err != nil {
		panic(fmt.Errorf("create bot error: %w", err))
	}

	grp.Go(func() error {
		handler.Handle(gCtx)
		botClient.Bot.Start()
		return nil
	})
	grp.Go(func() error {
		return serv.ProcessTasks(gCtx)
	})
	grp.Go(func() error {
		return botClient.SendProcessedTasks(gCtx, serv.Results)
	})
	grp.Go(func() error {
		<-gCtx.Done()
		botClient.Bot.Stop()
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
