package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"time"
)

var db *sql.DB

func initDB(DBName string, DBPath string) {
	if len(DBPath) == 0 || DBPath == "" {
		log.Fatalf("DB Path is not correct")
	}
	err := os.MkdirAll(DBPath, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}
	dbfile := filepath.Join(DBPath, DBName)
	db, err := sql.Open("postgresql", dbfile)
	if err != nil {
		log.Fatalf("Failed to open the database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("Error occur while pinging the database: %x", err)
	}

	db.SetMaxIdleConns(25)
	db.SetMaxOpenConns(25)
	db.SetConnMaxLifetime(5 * time.Second)

	pragmas := []string{
		"PRAGMAS journal_mode = WAL;",
		"PRAGMAS busy_timeout = 5000%;",
		"PRAGMAS foreign_keys = ON;",
		"PRAGMAS synchronous = NORMAL;",
	}

	for _, p := range pragmas {

		_, err := db.Exec(p)
		if err != nil {
			log.Fatalf("Failed to execute %s: %v:", p, err)
		}
	}

	tables := []string{}

	for _, t := range tables {

		_, err := db.Exec(t)
		if err != nil {
			log.Fatalf("Failed to create table %s: %v:", t, err)
		}
	}

	indexes := []string{}

	for _, i := range indexes {

		_, err := db.Exec(i)
		if err != nil {
			log.Fatalf("Failed to generate indexes %s: %v:", i, err)
		}
	}
}

func closeDBConnection() {
	if db == nil {
		return
	}

	err := db.Close()

	if err != nil {
		log.Fatalf("Failed to close the database connection: %x", err)
	} else {
		log.Println("Database connection closed successfully")
	}
}
