package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Elissbar/meeting-summary-bot/internal/gigachat"
	"github.com/Elissbar/meeting-summary-bot/internal/models"
	"github.com/Elissbar/meeting-summary-bot/internal/salutespeech"
	"github.com/Elissbar/meeting-summary-bot/internal/storage"
	tg "gopkg.in/telebot.v3"
)

type Service struct {
	ctx     context.Context
	salute  *salutespeech.SaluteSpeechClient
	giga    *gigachat.GigaChatClient
	storage *storage.DBStorage
	wg      *sync.WaitGroup
	Bot     *tg.Bot
	// Каналы придется закрывать в Graceful Shutdown
	Tasks   chan models.Meeting
	Results chan models.Meeting
	// Timeouts
	waitPlaceInChan  time.Duration
	stopProcessTasks time.Duration
}

func NewService(
	ctx context.Context,
	salute *salutespeech.SaluteSpeechClient,
	giga *gigachat.GigaChatClient,
	storage *storage.DBStorage,
	bot *tg.Bot,
	wg *sync.WaitGroup,
	waitPlace, stopProcess time.Duration,
) *Service {
	s := &Service{
		ctx, salute, giga, 
		storage, wg, bot, 
		make(chan models.Meeting, 100), make(chan models.Meeting, 100), 
		waitPlace, stopProcess,
	}
	go func() {
		if err := s.ProcessTasks(); err != nil {
			fmt.Printf("Order processor stopped with error: %v\n", err)
		}
	}()
	return s
}

func (s *Service) CreateTask(fileID string, userID int64) (int64, error) {
	ctx, cancel := context.WithTimeout(s.ctx, time.Second*10)
	defer cancel()

	meetingID, err := s.storage.CreateTask(ctx, userID, fileID)
	if err != nil {
		return -1, err
	}

	return meetingID, nil
}

func (s *Service) ProcessTasks() error {
	numWorkers := 5
	s.wg.Add(numWorkers)
	for i := range numWorkers {
		go s.worker(i)
	}

	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	for range ticker.C {

		// Выходим, еслы получили сигнал остановки
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}

		ctx, cancel := context.WithTimeout(s.ctx, time.Second*5)
		defer cancel()

		rows, err := s.storage.GetAllNewTasks(ctx)
		if err != nil {
			return err
		}

		for _, row := range rows {
			select {
			case s.Tasks <- row:
			case <-time.After(s.waitPlaceInChan):
				select {
				case s.Tasks <- row:
				case <-time.After(s.stopProcessTasks):
					fmt.Printf("Tasks channel full, skipping task %d\n", row.ID)
				}
			}
		}
	}

	return nil
}

func (s *Service) worker(workerID int) error {
	return nil
}

// 	for task := range s.Tasks {

// 		fileID, err := s.salute.Send(file)
// 		if err != nil {
// 			return -1, err
// 		}

// 		createdTask, err := s.salute.StartProcess(fileID.Result.RequestFileID)
// 		if err != nil {
// 			return -1, err
// 		}

// 		fileID, err := s.salute.Send(file)
// 		if err != nil {
// 			return -1, err
// 		}

// 		createdTask, err := s.salute.StartProcess(fileID.Result.RequestFileID)
// 		if err != nil {
// 			return -1, err
// 		}
// 		taskStatus := task.Status
// 		// var task models.SaluteTaskResponse

// 		for !slices.Contains([]string{"CANCELED", "DONE", "ERROR"}, taskStatus) {
// 			time.Sleep(time.Millisecond * 500)
// 			taskResponse, err := s.salute.CheckTask(task.taskID)
// 			if err != nil {
// 				return err
// 			}

// 			taskStatus = taskResponse.Result.Status
// 		}

// 		var transcription, chatResponse string
// 		if taskStatus == "DONE" {
// 			transcription, err := s.salute.DownloadFile(task.requestFileID)
// 			if err != nil {
// 				return err
// 			}

// 			chatResponse, err = s.giga.Send(transcription)
// 			if err != nil {
// 				return err
// 			}
// 		}

// 		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
// 		defer cancel()
// 		s.storage.UpdateTasks(ctx, task.userID, task.requestFileID, transcription, chatResponse, taskStatus)
// 	}
// 	return nil
