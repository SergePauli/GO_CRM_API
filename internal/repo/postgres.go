package repo

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func NewPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return db, nil
}
