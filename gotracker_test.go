package gotracker

import (
  "fmt"
  "github.com/tumdum/bencoding"
  "io/ioutil"
  "net/http"
  "net/http/httptest"
  "net/url"
  "testing"
)

const (
  defaultHost     = "http://example.com/"
  defaultInfoHash = "abcdeabcdeabcdeabcde"
  otherInfoHash   = "aaaaaaaaaaaaaaaaaaaa"
  defaultPeerId   = "01234567890123456789"
  otherPeerId     = "00000000000000000000"
  defaultIp       = "127.0.0.1"
  defaultPort     = 2048
  emptyResponse   = "d8:intervali1800e5:peerslee"
)

func TestExtractRequestDataWillReturnErrorOnNil(t *testing.T) {
  r, e := extractRequestData(nil)
  if r != nil || e == nil {
    t.Fatalf("Expected r nil and e not nil, got (%#v,%#v)", r, e)
  }
}

func defaultSeederFormValues() url.Values {
  opts := make(url.Values)
  opts["info_hash"] = []string{defaultInfoHash}
  opts["peer_id"] = []string{defaultPeerId}
  opts["port"] = []string{fmt.Sprint(defaultPort)}
  opts["uploaded"] = []string{"0"}
  opts["downloaded"] = []string{"0"}
  opts["left"] = []string{"0"}
  opts["event"] = []string{"started"}
  return opts
}

func newGetRequest(baseUrl string, formValues url.Values) *http.Request {
  req, err := http.NewRequest("GET", baseUrl+"?"+formValues.Encode(), nil)
  if err != nil {
    panic(err)
  }
  req.RemoteAddr = defaultIp + ":4321"
  req.ParseForm()
  return req
}

func TestRequestShouldSucceedWithCorrectRequest(t *testing.T) {
  req := newGetRequest(defaultHost, defaultSeederFormValues())
  data, err := extractRequestData(req)
  if err != nil {
    t.Fatalf("extractRequestData should not fail, err: %v, req: '%v'", err, req)
  }
  if data.Ip != "127.0.0.1" || data.InfoHash != defaultInfoHash || data.PeerId != defaultPeerId || data.Port != defaultPort {
    t.Fatalf("Expected correctly extracted data, got '%#v'", data)
  }
}

func defaultGetRequestWithRemoved(fieldToRemove string) *http.Request {
  formValues := defaultSeederFormValues()
  delete(formValues, fieldToRemove)
  return newGetRequest(defaultHost, formValues)
}

func TestRequestsWithMissingFieldsShouldFail(t *testing.T) {
  requests := []*http.Request{}
  requests = append(requests, defaultGetRequestWithRemoved("port"))
  requests = append(requests, defaultGetRequestWithRemoved("peer_id"))
  requests = append(requests, defaultGetRequestWithRemoved("info_hash"))
  for _, r := range requests {
    _, err := extractRequestData(r)
    if err == nil {
      t.Fatalf("extractRequestData should have failed")
    }
  }
}

func TestRequestWithBrokenRemoteAddressShouldFail(t *testing.T) {
  req := newGetRequest(defaultHost, defaultSeederFormValues())
  req.RemoteAddr = ""
  _, err := extractRequestData(req)
  if err == nil {
    t.Fatalf("extractRequestData should fail since remote address is broken: '%v'", req.RemoteAddr)
  }
}

func TestTrackerShouldReturnEmptyListOfPeersOnFirstRequest(t *testing.T) {
  r := newGetRequest(defaultHost, defaultSeederFormValues())
  tracker := MakeTracker(ioutil.Discard, 1800)
  recorder := httptest.NewRecorder()
  tracker.ServeHTTP(recorder, r)
  if recorder.Code != http.StatusOK {
    t.Fatalf("First correct response should be 200 OK")
  }
  if recorder.Body.String() != emptyResponse {
    t.Fatalf("Expected empty bencoded response, got '%v'", recorder.Body.String())
  }
}

func TestTrackerShouldReturnCorrectWaitInterval(t *testing.T) {
  intervals := []int{1, 2, 3, 4, 100, 200, 500, 1800, 5000, 40000}
  for _, interval := range intervals {
    req := newGetRequest(defaultHost, defaultSeederFormValues())
    tracker := MakeTracker(ioutil.Discard, interval)
    recorder := httptest.NewRecorder()
    tracker.ServeHTTP(recorder, req)
    if recorder.Code != http.StatusOK {
      t.Fatalf("Wrong status code: '%v'", recorder.Code)
    }
    expected := "d8:intervali" + fmt.Sprint(interval) + "e5:peerslee"
    if recorder.Body.String() != expected {
      t.Fatalf("Wrong body, expected '%v', got '%v'", expected, recorder.Body.String())
    }
  }
}

func performRequest(t *Tracker, peerId string, infoHash string) *httptest.ResponseRecorder {
  opts := defaultSeederFormValues()
  opts["peer_id"] = []string{peerId}
  opts["info_hash"] = []string{infoHash}
  req := newGetRequest(defaultHost, opts)
  rec := httptest.NewRecorder()
  t.ServeHTTP(rec, req)
  return rec
}

func TestTrackerShouldReturnNonemptyListOfPeersInSubsequentRequest(t *testing.T) {
  tracker := MakeTracker(ioutil.Discard, 1800)
  performRequest(tracker, defaultPeerId, defaultInfoHash)
  rec := performRequest(tracker, "aaaaabbbbbcccccddddd", defaultInfoHash)

  resp := make(map[string]interface{})
  bencoding.Unmarshal(rec.Body.Bytes(), &resp)
  if rinterval := resp["interval"].(int64); rinterval != 1800 {
    t.Fatalf("Expected interval 1800, got '%v'", rinterval)
  }
  peer := resp["peers"].([]interface{})[0].(map[string]interface{})
  if peer["id"].(string) != defaultPeerId || peer["ip"].(string) != defaultIp || peer["port"].(int64) != defaultPort {
    t.Fatalf("Incorrect peer found: '%v'", peer)
  }
}

func TestRequestsForDifferentInfoHashesShouldBeUnrelated(t *testing.T) {
  tracker := MakeTracker(ioutil.Discard, 1800)
  performRequest(tracker, defaultPeerId, defaultInfoHash)

  rec := performRequest(tracker, defaultPeerId, otherInfoHash)
  if rec.Body.String() != emptyResponse {
    t.Fatalf("Expected empty response, got '%v'", rec.Body.String())
  }

  rec2 := performRequest(tracker, otherPeerId, "aaaaabbbbbcccccddddd")
  if rec2.Body.String() != emptyResponse {
    t.Fatalf("Expected empty response, got '%v'", rec2.Body.String())
  }
}
