package main

import (
	"bufio"
	"context"
	"database/sql"
	"io"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/pflag"
	"golang.org/x/exp/slog"
)

type nullHandler struct{}

func (h *nullHandler) Enabled(context.Context, slog.Level) bool {
	return false
}
func (h *nullHandler) Handle(context.Context, slog.Record) error {
	return nil
}
func (h *nullHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}
func (h *nullHandler) WithGroup(name string) slog.Handler {
	return h
}

func main() {
	debugLog := pflag.Bool("debug-log", false, "output debug logs to file")
	pflag.Parse()

	if *debugLog {
		f, err := os.OpenFile(
			"ltop_debug.log",
			os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
			0644,
		)
		if err != nil {
			panic(err.Error())
		}
		slog.SetDefault(
			slog.New(
				slog.NewTextHandler(
					f,
					&slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true},
				),
			),
		)
	} else {
		slog.SetDefault(slog.New(&nullHandler{}))
	}

	var err error
	var input *os.File
	if pflag.NArg() > 0 {
		filename := pflag.Arg(0)
		slog.Info("opening file", "filename", filename)
		input, err = os.Open(filename)
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
