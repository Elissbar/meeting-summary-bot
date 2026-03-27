package service

import (
	"io"
	"slices"
	"time"

	"github.com/Elissbar/meeting-summary-bot/internal/gigachat"
	"github.com/Elissbar/meeting-summary-bot/internal/models"
	"github.com/Elissbar/meeting-summary-bot/internal/salutespeech"
)

type task struct {
}

type Service struct {
	salute *salutespeech.SaluteSpeechClient
	giga   *gigachat.GigaChatClient
}

func NewService(salute *salutespeech.SaluteSpeechClient, giga *gigachat.GigaChatClient) *Service {
	s := &Service{salute, giga}
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

	parsedFile, err := s.salute.DownloadFile(task.Result.ResponseFileID)
	if err != nil {
		return "", err
	}

	_, err = s.giga.Send(parsedFile[0].Results[0].NormalizedText)
	if err != nil {
		return "", err
	}

	return fileID, nil
}
