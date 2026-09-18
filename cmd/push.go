package main

import (
	"net/http"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/listmonk/internal/push"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// firebaseWebConfig is the public Firebase web-app config served to browsers.
// These keys are public by design (they ship in client JS); the secret
// service-account key is never exposed here.
type firebaseWebConfig struct {
	APIKey            string `json:"api_key"`
	AuthDomain        string `json:"auth_domain"`
	ProjectID         string `json:"project_id"`
	MessagingSenderID string `json:"messaging_sender_id"`
	AppID             string `json:"app_id"`
	VAPIDKey          string `json:"vapid_key"`
}

// getFirebaseWebConfig reads the single source of truth for the web-push
// project from server env so the JS client, the service worker, and the Go
// FCM sender can never disagree about which Firebase project to use.
func getFirebaseWebConfig() firebaseWebConfig {
	projectID := firstEnv("FIREBASE_PROJECT_ID", "OMNIPOST_FIREBASE_PROJECT_ID")
	return firebaseWebConfig{
		APIKey:            firstEnv("FIREBASE_WEB_API_KEY", "OMNIPOST_FIREBASE_WEB_API_KEY"),
		AuthDomain:        firstEnv("FIREBASE_WEB_AUTH_DOMAIN", "OMNIPOST_FIREBASE_WEB_AUTH_DOMAIN"),
		ProjectID:         projectID,
		MessagingSenderID: firstEnv("FIREBASE_WEB_SENDER_ID", "OMNIPOST_FIREBASE_WEB_SENDER_ID"),
		AppID:             firstEnv("FIREBASE_WEB_APP_ID", "OMNIPOST_FIREBASE_WEB_APP_ID"),
		VAPIDKey:          firstEnv("FIREBASE_VAPID_KEY", "OMNIPOST_FIREBASE_VAPID_KEY"),
	}
}

// GetFirebaseConfig serves the public Firebase config for the admin JS client.
func (a *App) GetFirebaseConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, okResp{getFirebaseWebConfig()})
}

// ServeFirebaseSW serves /firebase-messaging-sw.js from the site root with
// the Firebase web config injected from server env. Web Push requires the
// service worker at the site root scope — it cannot live under /admin/.
func (a *App) ServeFirebaseSW(c echo.Context) error {
	b, err := a.fs.Read("/public/firebase-messaging-sw.js")
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "firebase-messaging-sw.js not found in static assets")
	}

	cfg := getFirebaseWebConfig()
	out := string(b)
	for key, val := range map[string]string{
		"__FIREBASE_API_KEY__":            cfg.APIKey,
		"__FIREBASE_AUTH_DOMAIN__":        cfg.AuthDomain,
		"__FIREBASE_PROJECT_ID__":         cfg.ProjectID,
		"__FIREBASE_MESSAGING_SENDER_ID__": cfg.MessagingSenderID,
		"__FIREBASE_APP_ID__":             cfg.AppID,
	} {
		out = strings.ReplaceAll(out, key, val)
	}

	return c.Blob(http.StatusOK, "application/javascript; charset=utf-8", []byte(out))
}

// SavePushToken registers (or refreshes) a browser FCM token.
// Any authenticated admin may register their own browser.
func (a *App) SavePushToken(c echo.Context) error {
	var req struct {
		Token string `json:"token"`
		Label string `json:"label"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" || len(req.Token) > stdInputMaxLen {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid FCM token")
	}
	if len(req.Label) > stdInputMaxLen {
		return echo.NewHTTPError(http.StatusBadRequest, "label too long")
	}

	var id int
	if err := a.queries.UpsertPushToken.Get(&id,
		uuid.Must(uuid.NewV4()).String(), req.Token, strings.TrimSpace(req.Label)); err != nil {
		a.log.Printf("error saving push token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error saving push token")
	}

	return c.JSON(http.StatusOK, okResp{map[string]int{"id": id}})
}

// GetPushTokens lists registered browser tokens.
func (a *App) GetPushTokens(c echo.Context) error {
	var out []models.PushToken
	if err := a.queries.GetPushTokens.Select(&out); err != nil {
		a.log.Printf("error fetching push tokens: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error fetching push tokens")
	}
	if out == nil {
		out = []models.PushToken{}
	}
	return c.JSON(http.StatusOK, okResp{out})
}

// DeletePushToken removes a registered browser token.
func (a *App) DeletePushToken(c echo.Context) error {
	if _, err := a.queries.DeletePushToken.Exec(getID(c)); err != nil {
		a.log.Printf("error deleting push token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error deleting push token")
	}
	return c.JSON(http.StatusOK, okResp{true})
}

// SendTestPush delivers a test notification via FCM. With no token in the
// request it fans out to all active tokens. Tokens FCM reports as
// UNREGISTERED / INVALID_ARGUMENT are marked inactive (cleanup).
func (a *App) SendTestPush(c echo.Context) error {
	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Token string `json:"token"`
	}
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)
	if req.Title == "" {
		req.Title = "OmniPost test"
	}
	if req.Body == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "notification body is required")
	}

	cl, err := push.NewFromEnv()
	if err != nil {
		// 503 with setup guidance (missing/invalid service-account key).
		return echo.NewHTTPError(http.StatusServiceUnavailable, err.Error())
	}

	data := map[string]string{"url": "/admin/"}

	// Single-token send (the requesting browser).
	if t := strings.TrimSpace(req.Token); t != "" {
		invalid, err := cl.Send(t, req.Title, req.Body, data)
		deactivated := 0
		if invalid {
			deactivated = 1
			_, _ = a.queries.DeactivatePushToken.Exec(t)
		}
		if err != nil {
			a.log.Printf("error sending test push: %v", err)
			return echo.NewHTTPError(http.StatusBadGateway, err.Error())
		}
		return c.JSON(http.StatusOK, okResp{map[string]any{
			"success_count": 1, "failure_count": 0, "deactivated": deactivated,
		}})
	}

	// Fan-out to all active tokens.
	var tokens []string
	if err := a.queries.GetActivePushTokens.Select(&tokens); err != nil {
		a.log.Printf("error fetching active push tokens: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "error fetching push tokens")
	}
	if len(tokens) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no active push tokens. Open the Notifications page in a browser and enable notifications first")
	}

	res, err := cl.SendMulticast(tokens, req.Title, req.Body, data)
	if err != nil {
		a.log.Printf("error sending test push: %v", err)
		return echo.NewHTTPError(http.StatusBadGateway, err.Error())
	}

	// Cleanup: deactivate tokens FCM reports as permanently invalid.
	for _, t := range res.Invalid {
		if _, err := a.queries.DeactivatePushToken.Exec(t); err != nil {
			a.log.Printf("error deactivating push token: %v", err)
		}
	}

	a.log.Printf("test push: %d success, %d failed, %d deactivated", res.SuccessCount, res.FailureCount, len(res.Invalid))
	return c.JSON(http.StatusOK, okResp{map[string]any{
		"success_count": res.SuccessCount,
		"failure_count": res.FailureCount,
		"deactivated":   len(res.Invalid),
	}})
}
