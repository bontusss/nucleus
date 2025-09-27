package database

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

func Connect(url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return db, nil
}
