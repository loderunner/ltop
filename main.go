package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"flag"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/exp/slog"
)

var logBuf bytes.Buffer
var logger *slog.Logger = slog.New(slog.NewTextHandler(&logBuf, nil))

func main() {
	var input *os.File
	var err error

	logger.Info("parsing arguments")

	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		logger.Info("opening file", "filename", args[0])
		input, err = os.Open(args[0])
		if err != nil {
			panic(err.Error())
		}
	} else {
		logger.Info("reading from standard input")
		input = os.Stdin
	}

	logger.Info("opening database connection")
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err.Error())
	}

	db, err := newDatabase(sqlDB)
	if err != nil {
		panic(err.Error())
	}

	go scanInput(input, db)

	logger.Info("running application")
	if err := newApplication(db).Run(); err != nil {
		panic(err)
	}
}

func scanInput(input *os.File, db *DB) {
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		logger.Info("read line from input", "length", len(scanner.Bytes()))
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
