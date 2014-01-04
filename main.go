package main

import (
  "log"
  "net"
  "net/http"
  "net/url"
  "os"
)

func makeLogger(filename string) *log.Logger {
  logfile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
  if err != nil {
    panic(err)
  }
  return log.New(logfile, "gotracker ", log.LstdFlags)
}

type Peer struct {
  PeerId string
  Ip     net.IP
  Port   int
}

type AnnounceCfg struct {
  Logger        *log.Logger
  NewPeers      chan Peer
  ExistingPeers chan []Peer
}

func AnnounceHandler(logger *log.Logger, w http.ResponseWriter, r *http.Request) {
  r.ParseForm()
  host, _, err := net.SplitHostPort(r.RemoteAddr)
  if err != nil {
    w.Write([]byte(err.Error()))
    return
  }
  peer_id := r.FormValue("peer_id")
  port := r.FormValue("port")
  info_hash := url.QueryEscape(r.FormValue("info_hash"))
  logger.Printf("Will record %s %s:%s for info_hash: %v", peer_id, host, port, info_hash)
}

func main() {
  logger := makeLogger("debug.log")
  logger.Println("Starting")
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    AnnounceHandler(logger, w, r)
  })
  http.ListenAndServe(":8080", nil)
  defer logger.Println("Stopping")
}
