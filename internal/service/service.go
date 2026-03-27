package service

import (
	"io"
	"slices"
	"time"

	"github.com/Elissbar/meeting-summary-bot/internal/models"
	"github.com/Elissbar/meeting-summary-bot/internal/salutespeech"
)

type task struct {
}

type Service struct {
	salute *salutespeech.SaluteSpeechClient
}

func NewService(salute *salutespeech.SaluteSpeechClient) *Service {
	s := &Service{salute}
	return s
}

func (s *Service) UploadAndStartProcess(file io.ReadCloser) (string, error) {
	fileID, err := s.salute.Send(file)
	if err != nil {
		return "", err
	}

	createdTask, err := s.salute.StartProcess(fileID)
	if err != nil {
		return "", err
	}

	var taskStatus string
	var task models.SaluteTaskResponse
	for !slices.Contains([]string{"CANCELED", "DONE", "ERROR"}, taskStatus) {
		time.Sleep(time.Second * 1)
		task, err = s.salute.CheckTask(createdTask.Result.ID)
		if err != nil {
			return "", err
		}

		taskStatus = task.Result.Status
	}

	_, err = s.salute.DownloadFile(task.Result.ResponseFileID)
	if err != nil {
		return "", err
	}

	return fileID, nil
}
