// Package webhooks delivers signed lifecycle events (e.g. subscription
// confirmations) to per-list webhook targets so external apps can react in
// real time. Deliveries are at-least-once: every event carries an idempotency
// key and receivers must upsert on it. Dispatch is asynchronous and never
// fails the caller; failures are only logged.
package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// EventSubscriberConfirmed is fired when a subscriber confirms.
	EventSubscriberConfirmed = "subscriber.confirmed"

	// SignatureHeader carries the hex HMAC-SHA256 of "<timestamp>.<body>".
	SignatureHeader = "X-Patra-Signature"
	// TimestampHeader carries the Unix signing time for replay protection.
	TimestampHeader = "X-Patra-Timestamp"

	maxBodyLog = 4096
)

// Target is one delivery destination.
type Target struct {
	ListID   int
	ListUUID string
	ListName string
	URL      string
	Secret   string
}

// Subscriber is the confirmed subscriber.
type Subscriber struct {
	UUID  string `json:"uuid"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// EventList is a confirmed list.
type EventList struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// Event is the webhook payload.
type Event struct {
	Event          string      `json:"event"`
	Test           bool        `json:"test,omitempty"`
	IdempotencyKey string      `json:"idempotency_key"`
	ConfirmedAt    string      `json:"confirmed_at"`
	Subscriber     Subscriber  `json:"subscriber"`
	Lists          []EventList `json:"lists"`
}

// Dispatcher delivers events.
type Dispatcher struct {
	Client     *http.Client
	Timeout    time.Duration
	MaxRetries int
	// RetryDelay is the base sleep between attempts (scaled by attempt #).
	RetryDelay time.Duration
	Log        *log.Logger
}

// NewDispatcher returns a Dispatcher with sane defaults.
func NewDispatcher(lo *log.Logger) *Dispatcher {
	if lo == nil {
		lo = log.New(io.Discard, "", 0)
	}
	return &Dispatcher{Timeout: 5 * time.Second, MaxRetries: 3, RetryDelay: time.Second, Log: lo}
}

// ValidateURL ensures a webhook target is an absolute http(s) URL.
// Plain http is only allowed for hosts that can never route on the public
// internet: loopback, single-label Docker-style DNS names (e.g. add-admin),
// and RFC1918/link-local IPs.
func ValidateURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !u.IsAbs() {
		return fmt.Errorf("webhook URL must be absolute (got %q)", raw)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("webhook URL must use http(s) (got %q)", raw)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("webhook URL has no host (got %q)", raw)
	}
	if u.Scheme == "http" && !isPrivateHost(u.Hostname()) {
		return fmt.Errorf("webhook URL must use https outside private networks (got %q)", raw)
	}
	return nil
}

func isPrivateHost(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	// Single-label names (Docker DNS, mDNS, intranet hosts) never resolve publicly.
	if !strings.Contains(host, ".") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLoopback()
	}
	return false
}

// Sign returns the hex HMAC-SHA256 signature of "<timestamp>.<body>".
func Sign(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Send delivers one event to one target synchronously with retries.
func (d *Dispatcher) Send(t Target, ev Event) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("webhook %s: marshal event: %w", t.URL, err)
	}

	client := d.Client
	if client == nil {
		client = &http.Client{Timeout: d.Timeout}
	}

	delay := d.RetryDelay
	if delay < 0 {
		delay = 0
	}
	var lastErr error
	for attempt := 0; attempt <= d.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * delay)
		}

		ts := time.Now().Unix()
		req, err := http.NewRequest(http.MethodPost, t.URL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("webhook %s: build request: %w", t.URL, err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(TimestampHeader, strconv.FormatInt(ts, 10))
		req.Header.Set(SignatureHeader, Sign(t.Secret, ts, body))

		res, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("webhook %s: attempt %d: %w", t.URL, attempt+1, err)
			d.Log.Printf("confirm webhook to %s (list %s) failed: %v", t.URL, t.ListUUID, err)
			continue
		}

		respBody, _ := io.ReadAll(io.LimitReader(res.Body, maxBodyLog))
		res.Body.Close()

		if res.StatusCode >= 200 && res.StatusCode < 300 {
			d.Log.Printf("confirm webhook to %s (list %s) delivered: %s", t.URL, t.ListUUID, res.Status)
			return nil
		}

		lastErr = fmt.Errorf("webhook %s: attempt %d: unexpected status %s: %s", t.URL, attempt+1, res.Status, string(respBody))
		d.Log.Printf("confirm webhook to %s (list %s) failed: %s", t.URL, t.ListUUID, lastErr)
	}

	return lastErr
}

// Dispatch fans one event out to every target, each in its own goroutine.
// It never blocks and never returns delivery errors; failures are logged.
func (d *Dispatcher) Dispatch(targets []Target, ev Event) {
	for _, t := range targets {
		t := t
		go func() {
			// Recover defensively: a webhook must never crash the server.
			defer func() {
				if r := recover(); r != nil {
					d.Log.Printf("confirm webhook to %s (list %s) panicked: %v", t.URL, t.ListUUID, r)
				}
			}()
			if err := d.Send(t, ev); err != nil {
				d.Log.Printf("confirm webhook to %s (list %s) giving up: %v", t.URL, t.ListUUID, err)
			}
		}()
	}
}
