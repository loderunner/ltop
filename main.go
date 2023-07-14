package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
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

	stats, err := db.stats()
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Count: %d\n", stats.count)
	fmt.Printf("Min timestamp: %s\n", stats.minTimestamp.String())
	fmt.Printf("Max timestamp: %s\n", stats.maxTimestamp.String())
}
