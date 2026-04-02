package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
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
	storage storage.Repository
	config  *config.Config
	wg      *sync.WaitGroup
	bot     *bot.Bot
	// Каналы придется закрывать в Graceful Shutdown
	numWorkers int
	Tasks      chan models.Meeting
	Results    chan models.Meeting
	GigaTasks  chan models.Meeting
}

func NewService(
	salute *salutespeech.SaluteSpeechClient,
	giga *gigachat.GigaChatClient,
	storage storage.Repository,
	config *config.Config,
	wg *sync.WaitGroup,
	bot *bot.Bot,
) *Service {
	s := &Service{
		salute: salute, giga: giga,
		storage: storage, config: config, wg: wg, bot: bot, numWorkers: config.NumWorkers,
		Tasks: make(chan models.Meeting, 100), Results: make(chan models.Meeting, 100),
		GigaTasks: make(chan models.Meeting, config.NumWorkers),
	}
	return s
}

func (s *Service) CreateUser(ctx context.Context, userID int64) error {
	chCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	return s.storage.CreateUser(chCtx, userID)
}

func (s *Service) CheckUser(ctx context.Context, userID int64) (bool, error) {
	chCxt, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	return s.storage.CheckUser(chCxt, userID)
}

func (s *Service) CreateTask(ctx context.Context, fileID string, userID int64) (int64, error) {
	chCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	meetingID, err := s.storage.CreateTask(chCtx, userID, fileID)
	if err != nil {
		return -1, err
	}

	return meetingID, nil
}

func (s *Service) GetAllMeetings(ctx context.Context, userID int64) ([]string, error) {
	chCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	meetingsStr := make([]string, 0)

	for id, err := range s.storage.GetAllMeetings(chCtx, userID) {
		if err != nil {
			return nil, err
		}

		meetingsStr = append(meetingsStr, strconv.FormatInt(id, 10)) // Convert int to string
	}
	return meetingsStr, nil
}

func (s *Service) GetMeeting(ctx context.Context, rowID string) (string, error) {
	chCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	transcript, err := s.storage.GetMeeting(chCtx, rowID)
	if err != nil {
		return "", err
	}

	return transcript, nil
}

func (s *Service) FindTranscription(ctx context.Context, keyword string) (string, error) {
	chCtx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	transcript, err := s.storage.FindTranscription(chCtx, keyword)
	if err != nil {
		return "", err
	}

	return transcript, nil
}

func (s *Service) RequestToChat(prompt string) (string, error) {
	return s.giga.Send(prompt, true)
}

func (s *Service) ProcessTasks(ctx context.Context) error {
	s.wg.Add(s.numWorkers)
	for i := range s.numWorkers {
		ind := i
		go s.worker(ctx, ind)
	}

	// Запускаем клиент гиги на последовательную обработку в 1 поток
	go func() {
		for task := range s.GigaTasks {
			s.gigaProcessTasks(ctx, task)
		}
	}()

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
		fileData, audio_encoding, err := s.bot.GetFile(task.FileID)
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

func (s *Service) markAsFailed(ctx context.Context, task models.Meeting, errMessage string) error {
	// Если в ходе работы воркера произошла ошибка, отмечаем такие задачи в БД
	chCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	task.Status = "FAILED"
	s.Results <- task
	fmt.Println("Задача завершилась ошибкой: ", errMessage, "Отправили задачу в канал результатов: ", task)

	return s.storage.UpdateTasks(chCtx, task)
}

func (s *Service) markTaskInProgress(ctx context.Context, task models.Meeting) error {
	// Отмечаем задачи в обработке, чтобы не загружать их в канал при след. тике
	Ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	task.Status = "IN_PROGRESS"
	return s.storage.UpdateTasks(Ctx, task)
}

func (s *Service) gigaProcessTasks(ctx context.Context, task models.Meeting) (err error) {
	defer func() {
		if err != nil {
			s.markAsFailed(ctx, task, err.Error())
		}
	}()

	chatResponse, err := s.giga.Send(task.Transcript, false)
	if err != nil {
		return err
	}
	task.Summary = chatResponse

	chCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	err = s.storage.UpdateTasks(chCtx, task)
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

	for meeting, err := range s.storage.GetAllNewTasks(chCtx) {
		if err != nil && !errors.Is(err, myerrors.ErrNoRows) {
			fmt.Println("Ошибка при получении новых задач: ", err)
			return err
		}
		fmt.Println("Получили задачу из БД в статусе NEW. Задача: ", meeting)

		select {
		case s.Tasks <- meeting:
			fmt.Println("Отправили задачу в канал:", meeting)
		case <-time.After(s.config.WaitPlaceInChan):
			select {
			case s.Tasks <- meeting:
				fmt.Println("Отправили задачу в канал:", meeting)
			case <-time.After(s.config.StopProcess):
				fmt.Printf("Tasks channel full, skipping task %d\n", meeting.ID)
			}
		}
	}

	return nil
}
