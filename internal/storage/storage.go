package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Elissbar/meeting-summary-bot/internal/models"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	sq "github.com/Masterminds/squirrel"
)

type DBStorage struct {
	DB *sql.DB
	builder sq.StatementBuilderType
}

func NewDatabaseStorage(connectionData string) (*DBStorage, error) {
	db, err := sql.Open("postgres", connectionData)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	// ПЕРВОЕ: проверяем соединение с БД
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(db)
	storage := &DBStorage{db, psql}

	// ВТОРОЕ: применяем миграции
	if err := storage.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return storage, nil
}

// Migrate применяет миграции БД из папки migrations.
func (db *DBStorage) Migrate() error {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func (db *DBStorage) GetAllNewTasks(ctx context.Context) ([]models.Transcriptions, error) {
	var transcriptions []models.Transcriptions

	err := db.builder.
		Select("user_id", "request_file_id", "status").
		From("transcriptions").
		Where(sq.Eq{"status": "NEW"}).
		QueryRowContext(ctx).Scan(&transcriptions)
	if err != nil {
		return nil, err
	}
	return transcriptions, nil
}

func (db *DBStorage) SaveTask(ctx context.Context, userID int64, fileID, taskID string) (int64, error) {
	insertQuery := db.builder.
		Insert("transcriptions").
		Columns("user_id", "request_file_id", "task_id").
		Values(userID, fileID, taskID).
		Suffix("RETURNING id")
	
	var id int64
	err := insertQuery.QueryRowContext(ctx).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (db *DBStorage) UpdateTasks(
	ctx context.Context, 
	userID int64, 
	fileID string, 
	transcription, summary, status string,
) error {
	_, err := db.builder.
		Update("transcriptions").
		Set("transcript", transcription).
		Set("summary", summary).
		Set("status", status).
		Where(sq.Eq{"user_id": userID}).
		Where(sq.Eq{"request_file_id": fileID}).
		ExecContext(ctx)
	if err != nil {
		return err
	}

	return nil
}
