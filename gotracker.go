package gotracker

import (
	"errors"
	"github.com/tumdum/bencoding"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

const (
	defaultNumWant = 50
)

var (
	NilRequestError      = errors.New("nil request")
	MissingPortError     = errors.New("port is missing form request")
	MissingPeerIdError   = errors.New("peer_id is missing from request")
	MissingInfoHashError = errors.New("info_hash is missing from request")
	MalformedRemoteAddr  = errors.New("RemoteAddr seems to be broken")
)

type Peer struct {
	Ip     string `bencoding:"ip"`
	Port   int    `bencoding:"port"`
	PeerId string `bencoding:"id"`
}

type RequestData struct {
	Peer
	InfoHash string
	NumWant  *int
}

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
	ret := &RequestData{Peer: Peer{ip, port, peer_id}, InfoHash: info_hash}

	numWant, _ := strconv.Atoi(r.FormValue("numwant"))
	ret.NumWant = &numWant
	return ret, nil
}

type PeerSet map[Peer]bool

type Tracker struct {
	Interval        int
	logger          *log.Logger
	m               sync.Mutex
	managedTorrents map[string]PeerSet
}

func MakeTracker(logSink io.Writer, interval int) *Tracker {
	t := new(Tracker)
	t.logger = log.New(logSink, "gotracker ", log.LstdFlags)
	t.Interval = interval
	t.managedTorrents = make(map[string]PeerSet)
	return t
}

type TrackerResponse struct {
	Interval int    `bencoding:"interval"`
	Peers    []Peer `bencoding:"peers"`
}

func (t *Tracker) logAndFail(reason string, err error, w http.ResponseWriter) {
	t.logger.Printf("Tracker failed due to '%v' with reason '%v'", err, reason)
	_, err = w.Write([]byte(reason))
	if err != nil {
		t.logger.Printf("Failed to write response in logAndFail!")
	}
}

// Assumes that access to tracker has been made thread safe,
// ie. t.m is already locked.
func (t *Tracker) addPeer(req *RequestData) {
	if _, ok := t.managedTorrents[req.InfoHash]; !ok {
		t.managedTorrents[req.InfoHash] = make(PeerSet)
	}
	t.managedTorrents[req.InfoHash][req.Peer] = true
}

// Assumes that access to tracker has been made thread safe,
// ie. t.m is already locked.
func (t *Tracker) collectPeers(req *RequestData) []Peer {
	ret := []Peer{}
	max := defaultNumWant
	if req.NumWant != nil {
		max = *req.NumWant
	}
	peers := t.managedTorrents[req.InfoHash]
	found := 0
	for k, v := range peers {
		if !v || k == req.Peer {
			continue
		}
		ret = append(ret, k)
		found++
		if found == max {
			break
		}
	}
	return ret
}

// Records req.Peer as one interested in req.Info hash and return other
// peers interested in that torrent.
func (t *Tracker) prepareResponse(req *RequestData) TrackerResponse {
	t.m.Lock()
	defer t.m.Unlock()
	t.addPeer(req)
	return TrackerResponse{t.Interval, t.collectPeers(req)}
}

func (t *Tracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, err := extractRequestData(r)
	if err != nil {
		t.logAndFail("Incorrect request", err, w)
		return
	}
	resp := t.prepareResponse(req)
	b, _ := bencoding.Marshal(resp)
	_, err = w.Write(b)
	if err != nil {
		t.logger.Printf("Failed to write response due to: '%v'", err)
	}
}
