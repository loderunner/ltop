package main

import (
	"bufio"
	"database/sql"
	"flag"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rivo/tview"
)

func main() {
	var input *os.File
	var err error

	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		input, err = os.Open(args[0])
		if err != nil {
			panic(err.Error())
		}
	} else {
		input = os.Stdin
	}

	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err.Error())
	}

	db, err := newDatabase(sqlDB)
	if err != nil {
		panic(err.Error())
	}

	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
		err = db.appendLog(scanner.Bytes())
		if err != nil {
			panic(err.Error())
		}
	}

	err = scanner.Err()
	if err != nil {
		panic(err.Error())
	}

	logs, err := db.queryLogs(time.Time{}, time.Now())
	if err != nil {
		panic(err.Error())
	}

	table := tview.NewTable()
	table.SetBorder(false)
	table.SetSelectable(true, false)
	for i, log := range logs {
		table.SetCellSimple(i, 0, log.timestamp.String())
		table.SetCellSimple(i, 1, log.message)
	}

	if err := tview.NewApplication().SetRoot(table, true).Run(); err != nil {
		panic(err)
	}
}
