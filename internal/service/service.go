package service

import (
	"io"

	"github.com/Elissbar/meeting-summary-bot/internal/salutespeech"
)

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

	err = s.salute.StartProcess(fileID)
	if err != nil {
		return "", err
	}
	return fileID, nil
}
