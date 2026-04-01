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
	salute *salutespeech.SaluteSpeechClient,
	giga *gigachat.GigaChatClient,
	Storage *storage.DBStorage,
	config *config.Config,
	wg *sync.WaitGroup,
	bot *bot.Bot,
) *Service {
	s := &Service{
		salute: salute, giga: giga,
		Storage: Storage, config: config, wg: wg, Bot: bot,
		Tasks: make(chan models.Meeting, 100), Results: make(chan models.Meeting, 100),
	}
	s.numWorkers = config.NumWorkers
	s.GigaTasks = make(chan models.Meeting, s.numWorkers)
	// go func() { // Запускаем обработку задач
	// 	if err := s.ProcessTasks(); err != nil {
	// 		fmt.Printf("Order processor stopped with error: %v\n", err)
	// 	}
	// }()
	// go func() { // Прослушивание бота для отправки готовых задач TODO: придумать как передать сюда контекст
	// 	if err := bot.SendProcessedTasks(s.Results); err != nil {
	// 		fmt.Printf("Order processor stopped with error: %v\n", err)
	// 	}
	// }()
	return s
}

func (s *Service) CreateUser(ctx context.Context, userID int64) error {
	chCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	return s.Storage.CreateUser(chCtx, userID)
}

func (s *Service) CheckUser(ctx context.Context, userID int64) (bool, error) {
	chCxt, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	return s.Storage.CheckUser(chCxt, userID)
}

func (s *Service) CreateTask(ctx context.Context, fileID string, userID int64) (int64, error) {
	chCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	meetingID, err := s.Storage.CreateTask(chCtx, userID, fileID)
	if err != nil {
		return -1, err
	}

	return meetingID, nil
}

func (s *Service) ProcessTasks(ctx context.Context) error {
	s.wg.Add(s.numWorkers)
	for i := range s.numWorkers {
		ind := i
		go s.worker(ctx, ind)
	}

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for range ticker.C {
		// Выходим, еслы получили сигнал остановки
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if err := s.uploadTasksToChannel(ctx); err != nil {
			return err
		}
	}

	// Запускаем клиент гиги на последовательную обработку в 1 поток
	for task := range s.GigaTasks {
		if err := s.gigaProcessTasks(ctx, task); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) worker(ctx context.Context, workerID int) (err error) {
	fmt.Printf("Run worker with ID: %d.\n", workerID)
	defer s.wg.Done()

	for task := range s.Tasks {
		defer func() {
			if err != nil {
				s.markAsFailed(ctx, task, err.Error())
			}
		}()

		curStatus := task.Status
		if err := s.markTaskInProgress(ctx, task); err != nil {
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
		s.GigaTasks <- task
	}
	return nil
}

func (s *Service) markAsFailed(ctx context.Context, task models.Meeting, errMessage string) {
	// Если в ходе работы воркера произошла ошибка, отмечаем такие задачи в БД
	chCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	task.Status = "FAILED"
	s.Storage.UpdateTasks(chCtx, task)

	s.Results <- task
	fmt.Println("Задача завершилась ошибкой: ", errMessage, "Отправили задачу в канал результатов: ", task)
}

func (s *Service) markTaskInProgress(ctx context.Context, task models.Meeting) error {
	// Отмечаем задачи в обработке, чтобы не загружать их в канал при след. тике
	Ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	task.Status = "IN_PROGRESS"
	return s.Storage.UpdateTasks(Ctx, task)
}

func (s *Service) gigaProcessTasks(ctx context.Context, task models.Meeting) error {
	chatResponse, err := s.giga.Send(task.Transcript)
	if err != nil {
		return err
	}
	task.Summary = chatResponse

	chCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	err = s.Storage.UpdateTasks(chCtx, task)
	if err != nil {
		return err
	}

	s.Results <- task
	fmt.Println("Отправили задачу в канал результатов:", task)
	return nil
}

func (s *Service) uploadTasksToChannel(ctx context.Context) error {
	chCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	rows, err := s.Storage.GetAllNewTasks(chCtx)
	if err != nil && !errors.Is(err, myerrors.ErrNoRows) {
		fmt.Println("Ошибка при получении новых задач: ", err)
		return err
	}
	fmt.Println("Получили задачи из БД в статусе NEW. Кол-во:", len(rows))

	for _, row := range rows {
		// Если задача потеряется, она отправится в канал при след. тике
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
