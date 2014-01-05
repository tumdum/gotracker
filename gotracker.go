package gotracker

import (
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
  "github.com/tumdum/bencoding"
)

type Peer struct {
	Ip     string `bencoding:"ip"`
	Port   int    `bencoding:"port"`
	PeerId string `bencoding:"id"`
}

type RequestData struct {
	Peer
	InfoHash string
}

var (
	NilRequestError      = errors.New("nil request")
	MissingPortError     = errors.New("port is missing form request")
	MissingPeerIdError   = errors.New("peer_id is missing from request")
	MissingInfoHashError = errors.New("info_hash is missing from request")
	MalformedRemoteAddr  = errors.New("RemoteAddr seems to be broken")
)

// Important: assumes that was already called
func extractRequestData(r *http.Request) (*RequestData, error) {
	if r == nil {
		return nil, NilRequestError
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, MalformedRemoteAddr
	}
	port, err := strconv.Atoi(r.FormValue("port"))
	if err != nil {
		return nil, MissingPortError
	}
	peer_id := r.FormValue("peer_id")
	if peer_id == "" {
		return nil, MissingPeerIdError
	}
	info_hash := url.QueryEscape(r.FormValue("info_hash"))
	if info_hash == "" {
		return nil, MissingInfoHashError
	}
	return &RequestData{Peer{ip, port, peer_id}, info_hash}, nil
}

type PeerSet map[Peer]bool

type Tracker struct {
  interval int
	logger          *log.Logger
	m               sync.Mutex
	managedTorrents map[string]PeerSet
}

func MakeTracker(logSink io.Writer, interval int) *Tracker {
	t := new(Tracker)
	t.logger = log.New(logSink, "gotracker ", log.LstdFlags)
  t.interval = interval
	return t
}

type TrackerResponse struct {
  Interval int `bencoding:"interval"`
  Peers []Peer `bencoding:"peers"`
}

func (t *Tracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  resp := TrackerResponse{}
  resp.Interval = t.interval
  b, _ := bencoding.Marshal(resp)
  _, err := w.Write(b)
  if err != nil {
    t.logger.Printf("Failed to write response due to: '%v'", err)
  }
}
