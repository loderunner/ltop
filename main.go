package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"flag"
	"io"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/exp/slog"
)

var logBuf bytes.Buffer

func main() {
	var input *os.File
	var err error

	slog.SetDefault(
		slog.New(
			slog.NewTextHandler(
				&logBuf,
				&slog.HandlerOptions{Level: slog.LevelInfo},
			),
		),
	)
	slog.Info("parsing arguments")

	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		slog.Info("opening file", "filename", args[0])
		input, err = os.Open(args[0])
		if err != nil {
			panic(err.Error())
		}
	} else {
		slog.Info("reading from standard input")
		input = os.Stdin
	}

	slog.Info("opening database connection")
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err.Error())
	}

	db, err := newDatabase(sqlDB)
	if err != nil {
		panic(err.Error())
	}

	go scanInput(input, db)

	slog.Info("running application")
	if err := newApplication(db).Run(); err != nil {
		panic(err)
	}
}

func scanInput(input io.Reader, db *DB) {
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		slog.Info("read line from input", "length", len(scanner.Bytes()))
		err := db.appendLog(scanner.Bytes())
		if err != nil {
			panic(err.Error())
		}
	}

	err := scanner.Err()
	if err != nil {
		panic(err.Error())
	}
}
