package database

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/otel"
)

type Score struct {
	ID          int       `json:"ID"`
	Scenario    string    `json:"scenario"`
	Score       int       `json:"score"`
	ProcessedAt time.Time `json:"processed_at"`
}

type ScoresDatabase interface {
	Close() error
	CreateScore(ctx context.Context, scenario string, score int) error
	GetAllScores(ctx context.Context) ([]Score, error)
	UpdateScore(ctx context.Context, id int, scenario string, score int) error
}

type Queries struct {
	ScoresDatabase
	db *sql.DB
}

func InitDB(databaseSourceName string) (*Queries, error) {
	db, err := sql.Open("sqlite3", databaseSourceName)
	if err != nil {
		return nil, err
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS scores (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"scenario" TEXT,
		"score" INTEGER,
		"processed_at" DATETIME
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, err
	}

	return &Queries{db: db}, nil
}

func (q *Queries) Close() error {
	return q.db.Close()
}

func (q *Queries) CreateScore(ctx context.Context, scenario string, score int) error {
	tracer := otel.Tracer("database")
	_, span := tracer.Start(ctx, "CreateScore")
	defer span.End()

	query := `INSERT INTO scores(scenario, score, processed_at) VALUES (?, ?, ?)`
	stmt, err := q.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(scenario, score, time.Now())
	return err
}

func (q *Queries) GetAllScores(ctx context.Context) ([]Score, error) {
	tracer := otel.Tracer("database")
	_, span := tracer.Start(ctx, "GetAllScores")
	defer span.End()

	query := `SELECT id, scenario, score, processed_at FROM scores ORDER BY processed_at DESC`
	rows, err := q.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []Score
	for rows.Next() {
		var s Score
		if err := rows.Scan(&s.ID, &s.Scenario, &s.Score, &s.ProcessedAt); err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}
	return scores, nil
}

func (q *Queries) UpdateScore(ctx context.Context, id int, scenario string, score int) error {
	tracer := otel.Tracer("database")
	_, span := tracer.Start(ctx, "UpdateScore")
	defer span.End()

	stmt, err := q.db.Prepare("UPDATE scores SET scenario = ?, score = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(scenario, score, id)
	return err
}

func (q *Queries) DeleteScore(ctx context.Context, id int) error {
	tracer := otel.Tracer("database")
	_, span := tracer.Start(ctx, "DeleteScore")
	defer span.End()

	stmt, err := q.db.Prepare("DELETE FROM scores WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	return err
}
