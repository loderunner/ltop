package main

import (
	"bytes"
	"database/sql"
	"testing"
	"time"
)

const inputData = `{"level":"info","msg":"It's 20:34:11.244","timestamp":"2023-07-24T20:34:11.241+02:00"}
{"level":"error","msg":"It's 20:34:12.241","timestamp":"2023-07-24T20:34:12.241+02:00"}
{"level":"info","msg":"It's 20:34:13.241","timestamp":"2023-07-24T20:34:13.241+02:00"}
{"level":"info","msg":"It's 20:34:14.241","timestamp":"2023-07-24T20:34:14.241+02:00"}
{"level":"error","msg":"It's 20:34:15.241","timestamp":"2023-07-24T20:34:15.241+02:00"}`

func TestRace(t *testing.T) {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("unexpected error creating SQL database: %s", err)
	}

	db, err := newDatabase(sqlDB)
	if err != nil {
		t.Fatalf("unexpected error creating database: %s", err)
	}

	input := bytes.NewReader([]byte(inputData))

	go scanInput(input, db)
	db.queryLogs(time.Time{}, time.Now().UTC())
	go db.queryLogs(time.Time{}, time.Now().UTC())
}
