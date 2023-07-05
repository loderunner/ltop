package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

func main() {
	var in *os.File
	var err error

	flag.Parse()
	args := flag.Args()
	if len(args) > 0 {
		in, err = os.Open(args[0])
		if err != nil {
			panic(err.Error())
		}
	} else {
		in = os.Stdin
	}

	db, err := newDatabase()
	if err != nil {
		panic(err.Error())
	}

	scanner := bufio.NewScanner(in)

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
