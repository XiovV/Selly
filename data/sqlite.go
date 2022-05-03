package data

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(fileName string) *Repository {
	db, err := sqlx.Connect("sqlite3", fileName)
	if err != nil {
		log.Fatal(err)
	}

	db.MustExec(migrateSchema)

	return &Repository{db: db}
}
