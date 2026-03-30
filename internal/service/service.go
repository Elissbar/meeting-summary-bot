package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/Elissbar/meeting-summary-bot/internal/bot"
	"github.com/Elissbar/meeting-summary-bot/internal/config"
	myerrors "github.com/Elissbar/meeting-summary-bot/internal/errors"
	"github.com/Elissbar/meeting-summary-bot/internal/gigachat"
	"github.com/Elissbar/meeting-summary-bot/internal/models"
	"github.com/Elissbar/meeting-summary-bot/internal/salutespeech"
	"github.com/Elissbar/meeting-summary-bot/internal/storage"
)

type Service struct {
	Ctx     context.Context
	salute  *salutespeech.SaluteSpeechClient
	giga    *gigachat.GigaChatClient
	Storage *storage.DBStorage
	config  *config.Config
	wg      *sync.WaitGroup
	Bot     *bot.Bot
	// Каналы придется закрывать в Graceful Shutdown
	numWorkers int
	Tasks      chan models.Meeting
	Results    chan models.Meeting
	GigaTasks  chan models.Meeting
}

func NewService(
	Ctx context.Context,
	salute *salutespeech.SaluteSpeechClient,
	giga *gigachat.GigaChatClient,
	Storage *storage.DBStorage,
	config *config.Config,
	wg *sync.WaitGroup,
	bot *bot.Bot,
) *Service {
	s := &Service{
		Ctx: Ctx, salute: salute, giga: giga,
		Storage: Storage, config: config, wg: wg, Bot: bot,
		Tasks: make(chan models.Meeting, 100), Results: make(chan models.Meeting, 100),
	}
	s.numWorkers = config.NumWorkers
	s.GigaTasks = make(chan models.Meeting, s.numWorkers)
	go func() {
		if err := s.ProcessTasks(); err != nil {
			fmt.Printf("Order processor stopped with error: %v\n", err)
		}
	}()
	go func() {
		if err := bot.SendProcessedTasks(s.Results); err != nil {
			fmt.Printf("Order processor stopped with error: %v\n", err)
		}
	}()
	return s
}

func (s *Service) CreateUser(userID int64) error {
	Ctx, cancel := context.WithTimeout(s.Ctx, time.Second*10)
	defer cancel()

	return s.Storage.CreateUser(Ctx, userID)
}

func (s *Service) CheckUser(userID int64) (bool, error) {
	Ctx, cancel := context.WithTimeout(s.Ctx, time.Second*10)
	defer cancel()

	return s.Storage.CheckUser(Ctx, userID)
}

func (s *Service) CreateTask(fileID string, userID int64) (int64, error) {
	Ctx, cancel := context.WithTimeout(s.Ctx, time.Second*10)
	defer cancel()

	meetingID, err := s.Storage.CreateTask(Ctx, userID, fileID)
	if err != nil {
		return -1, err
	}

	return meetingID, nil
}

func (s *Service) ProcessTasks() error {
	go s.gigaProcessTasks()

	s.wg.Add(s.numWorkers)
	for i := range s.numWorkers {
		ind := i
		go s.worker(ind)
	}

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for range ticker.C {

		// Выходим, еслы получили сигнал остановки
		select {
		case <-s.Ctx.Done():
			return nil
		default:
		}

		if err := s.uploadTasksToChannel(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) worker(workerID int) (err error) {
	fmt.Printf("Run worker with ID: %d.\n", workerID)
	defer s.wg.Done()

	for task := range s.Tasks {
		// Если в ходе работы воркера произошла ошибка, отмечаем такие задачи в БД
		defer func() {
			if err != nil {
				Ctx, cancel := context.WithTimeout(s.Ctx, time.Second*5)
				defer cancel()
				s.Storage.UpdateTasks(Ctx, task, "FAILED")

				task.Status = "FAILED"
				s.Results <- task
				fmt.Println("Задача завершилась ошибкой: ", err, "Отправили задачу в канал результатов: ", task)
			}
		}()

		curStatus := task.Status
		if err := s.markTaskInProgress(task); err != nil {
			return err
		}

		fmt.Println("Воркер: ", workerID, "Берем данные из канала: ", task, "Статус задач:", task.Status)
		fileData, audio_encoding, err := s.Bot.GetFile(task.FileID)
		if err != nil {
			return err
		}

		uploadedFile, err := s.salute.Send(fileData)
		if err != nil {
			return err
		}

		createdTask, err := s.salute.StartProcess(uploadedFile.Result.RequestFileID, audio_encoding)
		if err != nil {
			return err
		}

		var taskResponse models.SaluteTaskResponse
		for !slices.Contains([]string{"CANCELED", "DONE", "ERROR"}, curStatus) {
			time.Sleep(time.Millisecond * 500)
			taskResponse, err = s.salute.CheckTask(createdTask.Result.ID)
			if err != nil {
				return err
			}
			curStatus = taskResponse.Result.Status
		}

		var transcription string
		if curStatus == "DONE" {
			transcription, err = s.salute.DownloadFile(taskResponse.Result.ResponseFileID)
			if err != nil {
				return err
			}
		}

		task.Status = curStatus
		task.Transcript = transcription
		s.GigaTasks <- task // Будем блокировать воркеры, если канал занят, но иначе не придумал, ведь Гига обрабатывает только в 1 поток.
	}
	return nil
}

func (s *Service) markTaskInProgress(task models.Meeting) error {
	Ctx, cancel := context.WithTimeout(s.Ctx, time.Second*3)
	defer cancel()
	return s.Storage.UpdateTasks(Ctx, task, "IN_PROGRESS")
}

func (s *Service) gigaProcessTasks() error {
	for task := range s.GigaTasks {
		chatResponse, err := s.giga.Send(task.Transcript)
		if err != nil {
			return err
		}
		task.Summary = chatResponse

		Ctx, cancel := context.WithTimeout(s.Ctx, time.Second*3)
		defer cancel()
		err = s.Storage.UpdateTasks(Ctx, task, task.Status)
		if err != nil {
			return err
		}

		s.Results <- task
		fmt.Println("Отправили задачу в канал результатов:", task)
	}
	return nil
}

func (s *Service) uploadTasksToChannel() error {
	Ctx, cancel := context.WithTimeout(s.Ctx, time.Second*5)
	defer cancel()

	rows, err := s.Storage.GetAllNewTasks(Ctx)
	if err != nil && !errors.Is(err, myerrors.ErrNoRows) {
		fmt.Println("Ошибка при получении новых задач: ", err)
		return err
	}
	fmt.Println("Получили задачи из БД в статусе NEW. Кол-во:", len(rows))
	// fmt.Println("Одна из задач: ", rows[0])

	for _, row := range rows {
		select {
		case s.Tasks <- row:
			fmt.Println("Отправили задачу в канал:", row)
		case <-time.After(s.config.WaitPlaceInChan):
			select {
			case s.Tasks <- row:
				fmt.Println("Отправили задачу в канал:", row)
			case <-time.After(s.config.StopProcess):
				fmt.Printf("Tasks channel full, skipping task %d\n", row.ID)
			}
		}
	}
	return nil
}