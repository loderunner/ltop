package main

import (
	"bufio"
	"database/sql"
	"flag"
	"os"

	_ "github.com/mattn/go-sqlite3"
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

	go scanInput(input, db)

	if err := newApplication(db).Run(); err != nil {
		panic(err)
	}
}

func scanInput(input *os.File, db DB) {
	scanner := bufio.NewScanner(input)

	for scanner.Scan() {
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
