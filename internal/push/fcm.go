// Package push implements web-push delivery via Firebase Cloud Messaging
// (FCM HTTP v1 API) using a service-account key supplied through the
// FIREBASE_SERVICE_ACCOUNT_KEY env var (single-line JSON, never committed).
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

// Client sends FCM messages.
type Client struct {
	projectID string
	http      *http.Client
}

// Result summarizes a multicast send.
type Result struct {
	SuccessCount int      `json:"success_count"`
	FailureCount int      `json:"failure_count"`
	Invalid      []string `json:"invalid_tokens,omitempty"`
}

type fcmMessage struct {
	Message struct {
		Token        string            `json:"token"`
		Notification map[string]string `json:"notification,omitempty"`
		Data         map[string]string `json:"data,omitempty"`
	} `json:"message"`
}

type fcmErrorResp struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// serviceAccountKey reads the JSON key from the environment.
func serviceAccountKey() ([]byte, string) {
	for _, k := range []string{"FIREBASE_SERVICE_ACCOUNT_KEY", "OMNIPOST_FIREBASE_SERVICE_ACCOUNT_KEY", "LISTMONK_FIREBASE_SERVICE_ACCOUNT_KEY"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return []byte(v), k
		}
	}
	return nil, ""
}

// NewFromEnv builds a Client from FIREBASE_SERVICE_ACCOUNT_KEY.
// It returns a descriptive error when the credential is missing so API
// handlers can surface setup guidance instead of a cryptic failure.
func NewFromEnv() (*Client, error) {
	raw, _ := serviceAccountKey()
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing FIREBASE_SERVICE_ACCOUNT_KEY: generate a service-account private key in Firebase Console (Project settings → Service accounts) and set it as a single-line JSON env var (see .env.sample)")
	}

	var parsed struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("invalid FIREBASE_SERVICE_ACCOUNT_KEY JSON: %v", err)
	}

	projectID := strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID"))
	if projectID == "" {
		projectID = strings.TrimSpace(os.Getenv("OMNIPOST_FIREBASE_PROJECT_ID"))
	}
	if projectID == "" {
		projectID = parsed.ProjectID
	}
	if projectID == "" {
		return nil, fmt.Errorf("FIREBASE_SERVICE_ACCOUNT_KEY has no project_id and FIREBASE_PROJECT_ID is unset")
	}

	ts, err := tokenSourceFromKey(context.Background(), raw)
	if err != nil {
		return nil, err
	}

	return &Client{
		projectID: projectID,
		http: &http.Client{
			Timeout:   15 * time.Second,
			Transport: &authTransport{base: http.DefaultTransport, src: ts},
		},
	}, nil
}

// Send delivers one notification. It reports invalid=true when FCM says the
// registration token is no longer usable (UNREGISTERED / INVALID_ARGUMENT)
// so callers can deactivate it.
func (c *Client) Send(token, title, body string, data map[string]string) (invalid bool, err error) {
	var m fcmMessage
	m.Message.Token = token
	if title != "" || body != "" {
		m.Message.Notification = map[string]string{"title": title, "body": body}
	}
	if len(data) > 0 {
		m.Message.Data = data
	}

	b, err := json.Marshal(m)
	if err != nil {
		return false, err
	}

	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", c.projectID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return false, nil
	}

	var fe fcmErrorResp
	if err := json.Unmarshal(respBody, &fe); err == nil {
		st := strings.ToUpper(fe.Error.Status)
		if st == "UNREGISTERED" || st == "INVALID_ARGUMENT" || fe.Error.Code == 404 {
			return true, fmt.Errorf("fcm rejected token: %s", fe.Error.Message)
		}
		return false, fmt.Errorf("fcm error %d %s: %s", fe.Error.Code, fe.Error.Status, fe.Error.Message)
	}

	return false, fmt.Errorf("fcm error: HTTP %d: %s", resp.StatusCode, string(respBody))
}

// SendMulticast delivers to many tokens, collecting per-token outcomes.
// Invalid tokens are returned so callers can mark them inactive.
func (c *Client) SendMulticast(tokens []string, title, body string, data map[string]string) (Result, error) {
	var r Result
	for _, t := range tokens {
		if strings.TrimSpace(t) == "" {
			continue
		}
		invalid, err := c.Send(t, title, body, data)
		if err == nil {
			r.SuccessCount++
			continue
		}
		r.FailureCount++
		if invalid {
			r.Invalid = append(r.Invalid, t)
		} else {
			// Non-token error (auth, quota, network): stop and surface it
			// instead of burning through the remaining tokens.
			return r, err
		}
	}
	return r, nil
}
