package main

import (
  "log"
  "net"
  "net/http"
  "net/url"
  "os"
  "strconv"
  "sync"
)

func makeLogger(filename string) *log.Logger {
  logfile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
  if err != nil {
    panic(err)
  }
  return log.New(logfile, "gotracker ", log.LstdFlags)
}

type Peer struct {
  Host   string
  Port   int
  PeerId string
}

type NewPeersRequest struct {
  InfoHash string
  RespChan chan []Peer
}

type AnnounceCfg struct {
  Logger        *log.Logger
  NewPeers      chan RequestData
  ExistingPeers chan<- NewPeersRequest
}

type RequestData struct {
  Peer
  InfoHash string
}

func ExtractRequestData(r *http.Request) (*RequestData, error) {
  port, err := strconv.Atoi(r.FormValue("port"))
  if err != nil {
    return nil, err
  }
  host, _, err := net.SplitHostPort(r.RemoteAddr)
  if err != nil {
    return nil, err
  }
  peer_id := r.FormValue("peer_id")
  info_hash := url.QueryEscape(r.FormValue("info_hash"))
  return &RequestData{Peer{host, port, peer_id}, info_hash}, nil
}

func AnnounceHandler(cfg AnnounceCfg, w http.ResponseWriter, r *http.Request) {
  r.ParseForm()
  rdata, err := ExtractRequestData(r)
  if err != nil {
    cfg.Logger.Printf("Failed to extract request data due to: %s", err.Error())
    w.Write([]byte(err.Error()))
  }
  cfg.Logger.Printf("processing...")
  resp := make(chan []Peer)
  cfg.ExistingPeers <- NewPeersRequest{rdata.InfoHash, resp}
  peers := <-resp
  cfg.Logger.Printf("Received set of peers: %v", peers)
  cfg.NewPeers <- *rdata
}

var m sync.Mutex

type PeerSet map[Peer]bool

var peersmap map[string]PeerSet = make(map[string]PeerSet)

func peerConsumer(logger *log.Logger, peers chan RequestData) {
  for {
    select {
    case peer := <-peers:
      logger.Printf("Will record peer %v for info_hash %v", peer.Peer, peer.InfoHash)
      m.Lock()
      if _, ok := peersmap[peer.InfoHash]; !ok {
        peersmap[peer.InfoHash] = make(PeerSet)
      }
      peersmap[peer.InfoHash][peer.Peer] = true
      m.Unlock()
    }
  }
}

func peersProducer(logger *log.Logger, requests chan NewPeersRequest) {
  for {
    select {
    case req := <-requests:
      logger.Printf("Will produce peers for request %v", req)
      m.Lock()
      if _, ok := peersmap[req.InfoHash]; !ok {
        peersmap[req.InfoHash] = make(PeerSet)
      }

      size := 10
      if size > len(peersmap[req.InfoHash]) {
        size = len(peersmap[req.InfoHash])
      }
      ret := make([]Peer, size)
      c := 0
      if _, ok := peersmap[req.InfoHash]; !ok {
        req.RespChan <- []Peer{}
      } else {
        for k, ok := range peersmap[req.InfoHash] {
          if ok && c < size {
            ret[c] = k
            c++
          }
        }
        req.RespChan <- ret
      }
      m.Unlock()
    }
  }
}

func main() {
  logger := makeLogger("debug.log")
  logger.Println("Starting")

  newPeers := make(chan RequestData)
  requests := make(chan NewPeersRequest)
  go peerConsumer(logger, newPeers)
  go peersProducer(logger, requests)
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    AnnounceHandler(AnnounceCfg{logger, newPeers, requests}, w, r)
  })
  http.ListenAndServe(":8080", nil)
  defer logger.Println("Stopping")
}
