package main

import (
	"code.google.com/p/gcfg"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/tumdum/gotracker"
	"html/template"
	"net/http"
	"os"
	"time"
)

const (
	logFilePath     = "debug.log"
	defaultInterval = 1801
)

const (
	welcomeTemplateStr = `
	<html>
		<head></head>
		<body>
			<a href="list">List of all tracked files</a>
		</body>
	</html>`
)

var welcomeTemplate *template.Template

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

func BuildServer(tracker *gotracker.Tracker, cfg *gotracker.Config) *http.Server {
	m := mux.NewRouter()
	announceHandler := func(w http.ResponseWriter, r *http.Request) {
		tracker.ServeHTTP(w, r)
	}
	m.Path("/announce").HandlerFunc(announceHandler)

	listHandler := func(w http.ResponseWriter, r *http.Request) {
		tracker.ListAll(w, r)
	}
	m.Methods("GET").Path("/list").HandlerFunc(listHandler)

	infoHandler := func(w http.ResponseWriter, r *http.Request) {
		tracker.Info(w, r)
	}
	m.Methods("GET").Path("/info/{hash}").HandlerFunc(infoHandler)

	m.Methods("GET").Path("/").HandlerFunc(WelcomeHandler)

	return &http.Server{
		Addr:           cfg.Network.Host + ":" + fmt.Sprint(cfg.Network.Port),
		Handler:        m,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

func WelcomeHandler(w http.ResponseWriter, r *http.Request) {
	welcomeTemplate.Execute(w, nil)
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
	welcomeTemplate =
		template.Must(template.New("welcome").Parse(welcomeTemplateStr))
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
