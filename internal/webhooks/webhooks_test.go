package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func testDispatcher() *Dispatcher {
	d := NewDispatcher(log.New(io.Discard, "", 0))
	d.RetryDelay = time.Millisecond
	return d
}

func testEvent() Event {
	return Event{
		Event:          EventSubscriberConfirmed,
		IdempotencyKey: "sub-1:list-1",
		ConfirmedAt:    "2026-09-29T00:00:00Z",
		Subscriber:     Subscriber{UUID: "sub-1", Email: "a@example.com", Name: "A"},
		Lists:          []EventList{{UUID: "list-1", Name: "L1"}},
	}
}

func TestSign(t *testing.T) {
	body := []byte(`{"a":1}`)
	got := Sign("s3cr3t", 123, body)

	mac := hmac.New(sha256.New, []byte("s3cr3t"))
	mac.Write([]byte("123."))
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Fatalf("sign mismatch: got %s want %s", got, want)
	}
	if Sign("other", 123, body) == got {
		t.Fatal("different secret must give a different signature")
	}
}

func TestValidateURL(t *testing.T) {
	for _, tc := range []struct {
		url string
		ok  bool
	}{
		{"https://example.com/hook", true},
		{"https://example.com:8443/a?b=c", true},
		{"http://localhost:3000/hook", true},
		{"http://127.0.0.1/hook", true},
		{"http://add-admin:3000/api/webhooks/omnipost", true},
		{"http://192.168.1.10/hook", true},
		{"http://10.0.0.5:3000/hook", true},
		{"http://example.com/hook", false},
		{"http://8.8.8.8/hook", false},
		{"ftp://example.com/hook", false},
		{"notaurl", false},
		{"", false},
		{"/relative/path", false},
	} {
		if err := ValidateURL(tc.url); (err == nil) != tc.ok {
			t.Errorf("ValidateURL(%q) err=%v, want ok=%v", tc.url, err, tc.ok)
		}
	}
}

func TestSendSuccess(t *testing.T) {
	var (
		gotBody []byte
		gotSig  string
		gotTs   string
		gotCT   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotSig = r.Header.Get(SignatureHeader)
		gotTs = r.Header.Get(TimestampHeader)
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := testDispatcher()
	if err := d.Send(Target{ListUUID: "list-1", URL: srv.URL, Secret: "s3cr3t"}, testEvent()); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if gotCT != "application/json" {
		t.Errorf("content-type = %q", gotCT)
	}
	ts, err := strconv.ParseInt(gotTs, 10, 64)
	if err != nil || time.Since(time.Unix(ts, 0)) > time.Minute {
		t.Errorf("bad timestamp header %q", gotTs)
	}
	if want := Sign("s3cr3t", ts, gotBody); want != gotSig {
		t.Error("signature does not verify against received body")
	}
	var ev Event
	if err := json.Unmarshal(gotBody, &ev); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if ev.Event != EventSubscriberConfirmed || ev.Subscriber.Email != "a@example.com" || len(ev.Lists) != 1 {
		t.Errorf("unexpected payload: %+v", ev)
	}
}

func TestSendRetryThenSuccess(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt64(&hits, 1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := testDispatcher()
	d.MaxRetries = 3
	if err := d.Send(Target{URL: srv.URL}, testEvent()); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if hits != 3 {
		t.Errorf("hits = %d, want 3", hits)
	}
}

func TestSendGivesUp(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	d := testDispatcher()
	d.MaxRetries = 2
	if err := d.Send(Target{URL: srv.URL}, testEvent()); err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if hits != 3 {
		t.Errorf("hits = %d, want 3 (initial + 2 retries)", hits)
	}
}

func TestDispatchIsolatesFailures(t *testing.T) {
	var okHits, badHits int64
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&okHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&badHits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()

	d := testDispatcher()
	d.MaxRetries = 1
	done := make(chan struct{})
	go func() {
		d.Dispatch([]Target{{ListUUID: "good", URL: ok.URL}, {ListUUID: "bad", URL: bad.URL}}, testEvent())
		close(done)
	}()

	select {
	case <-done:
		// Dispatch must return without waiting for deliveries.
	case <-time.After(5 * time.Second):
		t.Fatal("Dispatch blocked")
	}

	deadline := time.Now().Add(10 * time.Second)
	for (atomic.LoadInt64(&okHits) == 0 || atomic.LoadInt64(&badHits) < 2) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if atomic.LoadInt64(&okHits) != 1 {
		t.Errorf("good target hits = %d, want 1", okHits)
	}
	if atomic.LoadInt64(&badHits) != 2 {
		t.Errorf("bad target hits = %d, want 2 (initial + 1 retry)", badHits)
	}
}
