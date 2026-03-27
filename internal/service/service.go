package service

import (
	"context"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/Elissbar/meeting-summary-bot/internal/gigachat"
	"github.com/Elissbar/meeting-summary-bot/internal/salutespeech"
	"github.com/Elissbar/meeting-summary-bot/internal/storage"
)

type task struct {
	userID                        int64
	requestFileID, taskID, Status string
}

type Service struct {
	salute  *salutespeech.SaluteSpeechClient
	giga    *gigachat.GigaChatClient
	storage *storage.DBStorage
	// cfg *config.Config
	Tasks            chan task // Придется закрывать в Graceful Shutdown
	waitPlaceInChan  time.Duration
	stopProcessTasks time.Duration
}

func NewService(
	salute *salutespeech.SaluteSpeechClient,
	giga *gigachat.GigaChatClient,
	storage *storage.DBStorage,
	waitPlace, stopProcess time.Duration,
	// cfg *config.Config,
) *Service {
	s := &Service{salute, giga, storage, make(chan task, 100), waitPlace, stopProcess}
	return s
}

func (s *Service) UploadAndStartProcess(file io.ReadCloser, userID int64) (int64, error) {
	fileID, err := s.salute.Send(file)
	if err != nil {
		return -1, err
	}

	createdTask, err := s.salute.StartProcess(fileID.Result.RequestFileID)
	if err != nil {
		return -1, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	id, err := s.storage.SaveTask(ctx, userID, fileID.Result.RequestFileID, createdTask.Result.ID)
	if err != nil {
		return -1, err
	}

	return id, nil
}

func (s *Service) UpdateTaskStatus() error {
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

	for i := range 5 {
		go s.worker(i)
	}

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		rows, err := s.storage.GetAllNewTasks(ctx)
		cancel()
		if err != nil {
			return err
		}

		for _, row := range rows {
			select {
			case s.Tasks <- task{row.UserID, row.RequestFileID, row.TaskID, row.Status}:
			default:
				fmt.Printf("Tasks channel full, skipping task %d\n", row.ID)
			}
		}
	}

	return nil
}

func (s *Service) worker(workerID int) error {
	for task := range s.Tasks {
		taskStatus := task.Status
		// var task models.SaluteTaskResponse

		for !slices.Contains([]string{"CANCELED", "DONE", "ERROR"}, taskStatus) {
			time.Sleep(time.Millisecond * 500)
			taskResponse, err := s.salute.CheckTask(task.taskID)
			if err != nil {
				return err
			}

			taskStatus = taskResponse.Result.Status
		}

		var transcription, chatResponse string
		if taskStatus == "DONE" {
			transcription, err := s.salute.DownloadFile(task.requestFileID)
			if err != nil {
				return err
			}

			chatResponse, err = s.giga.Send(transcription)
			if err != nil {
				return err
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		s.storage.UpdateTasks(ctx, task.userID, task.requestFileID, transcription, chatResponse, taskStatus)
	}
	return nil
}
