package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	myerrors "github.com/Elissbar/meeting-summary-bot/internal/errors"
	"github.com/Elissbar/meeting-summary-bot/internal/models"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgerrcode"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
)

type DBStorage struct {
	DB      *sql.DB
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

func (db *DBStorage) CreateUser(ctx context.Context, userID int64) error {
	insertQuery := db.builder.
		Insert("users").
		Columns("user_id").
		Values(userID)

	_, err := insertQuery.QueryContext(ctx)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("user with ID - %d already exists.", userID)
		}
		return fmt.Errorf("create user error: %v", err)
	}
	return nil
}

func (db *DBStorage) CheckUser(ctx context.Context, userID int64) (bool, error) {
	var cnt int64

	err := db.builder.
		Select("COUNT(*)").
		From("users").
		Where(sq.Eq{"user_id": userID}).
		QueryRowContext(ctx).Scan(&cnt)
	if err != nil {
		return false, fmt.Errorf("get users count error: %w", err)
	}

	return cnt == 1, nil
}

func (db *DBStorage) GetAllMeetings(ctx context.Context, userID int64) ([]int64, error) {
	var meetingIDs []int64 = make([]int64, 0)

	rows, err := db.builder.
		Select("id").
		From("meetings").
		Where(sq.Eq{"user_id": userID}).
		QueryContext(ctx)
	if err != nil {
		return meetingIDs, fmt.Errorf("get all meetings error: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var id int64

		if err := rows.Scan(&id); err != nil {
			return meetingIDs, fmt.Errorf("scan meeting ID error: %w", err)
		}

		meetingIDs = append(meetingIDs, id)
	}

	if err := rows.Err(); err != nil {
		return meetingIDs, fmt.Errorf("error iterating rows: %v", err)
	}

	return meetingIDs, nil
}

func (db *DBStorage) GetMeeting(ctx context.Context, rowID string) (string, error) {
	var transcript string

	err := db.builder.
		Select("transcript").
		From("meetings").
		Where(sq.Eq{"id": rowID}).
		QueryRowContext(ctx).Scan(&transcript)
	if err != nil {
		return transcript, fmt.Errorf("scan row id error: %w", err)
	}

	return transcript, nil
}

func (db *DBStorage) FindTranscription(ctx context.Context, keyword string) (string, error) {
	var transcript string

	err := db.builder.
		Select("transcript").
		From("meetings").
		Where(sq.Expr("transcription_vector @@ to_tsquery('russian', ?)", keyword)).
		QueryRowContext(ctx).Scan(&transcript)
	if err != nil {
		return transcript, fmt.Errorf("scan row id error: %w", err)
	}

	return transcript, nil
}

func (db *DBStorage) CreateTask(ctx context.Context, userID int64, fileID string) (int64, error) {
	insertQuery := db.builder.
		Insert("meetings").
		Columns("user_id", "file_id").
		Values(userID, fileID).
		Suffix("RETURNING id")

	var lastInsertID int64
	err := insertQuery.QueryRowContext(ctx).Scan(&lastInsertID)
	if err != nil {
		return 0, fmt.Errorf("scan row id error: %w", err)
	}

	return lastInsertID, nil
}

func (db *DBStorage) GetAllNewTasks(ctx context.Context) ([]models.Meeting, error) {
	var meetings []models.Meeting

	rows, err := db.builder.
		Select("id", "user_id", "file_id", "status").
		From("meetings").
		Where(sq.Eq{"status": "NEW"}).
		QueryContext(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return meetings, myerrors.ErrNoRows
		}
		return meetings, fmt.Errorf("get all NEW tasks error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var meeting models.Meeting

		err := rows.Scan(
			&meeting.ID,
			&meeting.UserID,
			&meeting.FileID,
			&meeting.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task row error: %w", err)
		}

		meetings = append(meetings, meeting)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return meetings, nil
}

func (db *DBStorage) UpdateTasks(
	ctx context.Context,
	task models.Meeting,
	status string,
	// userID int64,
	// fileID string,
	// transcription, summary, status string,
) error {
	_, err := db.builder.
		Update("meetings").
		Set("transcript", task.Transcript).
		Set("summary", task.Summary).
		Set("status", status).
		Set("transcription_vector", sq.Expr("to_tsvector('russian', ?)", task.Transcript)).
		Where(sq.Eq{"user_id": task.UserID}).
		Where(sq.Eq{"file_id": task.FileID}).
		ExecContext(ctx)
	if err != nil {
		return err
	}

	return nil
}
