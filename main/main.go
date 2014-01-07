package main

import (
	"code.google.com/p/gcfg"
	"fmt"
	"github.com/tumdum/gotracker"
	"net/http"
	"os"
	"time"
)

const (
	logFilePath     = "debug.log"
	defaultInterval = 1801
)

func BuildTracker(cfg *gotracker.Server) *gotracker.Tracker {
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		panic(err)
	}
	tracker := gotracker.MakeTracker(logFile, cfg)
	if tracker == nil {
		fmt.Errorf("failed to create tracker object")
	}
	return tracker
}

func BuildServer(handler http.Handler, cfg *gotracker.Config) *http.Server {
	return &http.Server{
		Addr:           cfg.Network.Host + ":" + fmt.Sprint(cfg.Network.Port),
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

func ReadConfig() (*gotracker.Config, error) {
	var cfg gotracker.Config
	err := gcfg.ReadFileInto(&cfg, "gotrackerrc")
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	cfg, err := ReadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	tracker := BuildTracker(&cfg.Server)
	server := BuildServer(tracker, cfg)
	tracker.Logger.Printf("Starting on %v:%v", cfg.Network.Host, cfg.Network.Port)
	server.ListenAndServe()
}
