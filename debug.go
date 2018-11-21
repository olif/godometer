package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
)

var logger = log.New(ioutil.Discard, "", 0)

func init() {
	logFilePath := flag.String("d", "default", "log file")
	flag.Parse()
	if *logFilePath == "" {
		return
	} else if *logFilePath == "default" {
		*logFilePath = "debug.txt"
	}

	f, err := os.OpenFile(*logFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Panic(err)
	}
	logger.SetOutput(f)
}

func debug(format string, a ...interface{}) {
	if a != nil {
		logger.Printf(format, a...)
	} else {
		logger.Println(format)
	}
}
