package main

import (
	"fmt"
	"github.com/tumdum/gotracker"
	"net/http"
	"os"
	"time"
)

const (
	logFilePath     = "debug.log"
	defaultInterval = 1800
)

func BuildTracker() *gotracker.Tracker {
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(err)
	}
	tracker := gotracker.MakeTracker(logFile, defaultInterval)
	if tracker == nil {
		fmt.Errorf("failed to create tracker object")
	}
	return tracker
}

func BuildServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:           "127.0.0.1:8080",
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

func main() {
	tracker := BuildTracker()
	server := BuildServer(tracker)
	server.ListenAndServe()
}
