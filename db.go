package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

func connect() *sql.DB {
	var (
		host     = "localhost"
		port     = 5432
		user     = "postgres"
		password = os.Getenv("DB_PASS")
		dbname   = os.Getenv("DB")
	)
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s", host, port, user, password, dbname)
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return db
}
