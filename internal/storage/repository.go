package storage

import (
	"context"
	"iter"
	"log/slog"

	"github.com/Elissbar/meeting-summary-bot/internal/models"
)

type Repository interface {
	UserRepository
	TaskRepository
	MeetingRepository
}

type UserRepository interface {
	CreateUser(ctx context.Context, userID int64) error
	CheckUser(ctx context.Context, userID int64) (bool, error)
}

type TaskRepository interface {
	CreateTask(ctx context.Context, userID int64, fileID string) (int64, error)
	GetAllNewTasks(ctx context.Context) iter.Seq2[models.Meeting, error]
	UpdateTasks(ctx context.Context, task models.Meeting) error
}

type MeetingRepository interface {
	GetAllMeetings(ctx context.Context, userID int64) iter.Seq2[int64, error]
	GetMeeting(ctx context.Context, rowID string) (string, error)
	FindTranscription(ctx context.Context, keyword string) (string, error)
}

func NewStorage(dbConnectionURI string, log *slog.Logger) (Repository, error) {
	return NewDatabaseStorage(dbConnectionURI, log)
}
