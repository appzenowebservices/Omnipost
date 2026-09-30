package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/appzenowebservices/patra/internal/auth"
	"github.com/appzenowebservices/patra/internal/webhooks"
	"github.com/appzenowebservices/patra/models"
	"github.com/labstack/echo/v4"
)

// dispatchConfirmWebhooks resolves webhook targets for the confirmed lists
// (per-list targets, plus the global env fallback) and fans a signed event
// out asynchronously. It never fails the request; all errors are logged.
func (a *App) dispatchConfirmWebhooks(subUUID string, lists []models.List) {
	if len(lists) == 0 {
		return
	}

	sub, err := a.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		a.log.Printf("confirm webhook: cannot load subscriber %s: %v", subUUID, err)
		return
	}

	uuids := make([]string, 0, len(lists))
	for _, l := range lists {
		uuids = append(uuids, l.UUID)
	}

	hooks, err := a.core.GetListWebhooks(uuids)
	if err != nil {
		a.log.Printf("confirm webhook: cannot load list targets %v: %v", uuids, err)
		return
	}
	byUUID := map[string]models.ListWebhook{}
	for _, h := range hooks {
		byUUID[h.ListUUID] = h
	}

	now := time.Now().UTC().Format(time.RFC3339)
	d := webhooks.NewDispatcher(a.log)

	// Per-list targets, each receiving an event scoped to its own list.
	for _, l := range lists {
		h, ok := byUUID[l.UUID]
		if !ok {
			continue
		}
		ev := confirmEvent(sub, []models.List{l}, now, false)
		d.Dispatch([]webhooks.Target{{
			ListID:   l.ID,
			ListUUID: l.UUID,
			ListName: l.Name,
			URL:      h.URL,
			Secret:   h.Secret,
		}}, ev)
	}

	// Global env fallback receives one event covering all confirmed lists.
	if u := firstEnv("PATRA_CONFIRM_WEBHOOK_URL"); u != "" {
		if err := webhooks.ValidateURL(u); err != nil {
			a.log.Printf("confirm webhook: invalid PATRA_CONFIRM_WEBHOOK_URL: %v", err)
			return
		}
		ev := confirmEvent(sub, lists, now, false)
		d.Dispatch([]webhooks.Target{{
			URL:    u,
			Secret: firstEnv("PATRA_CONFIRM_WEBHOOK_SECRET"),
		}}, ev)
	}
}

func confirmEvent(sub models.Subscriber, lists []models.List, at string, test bool) webhooks.Event {
	uuids := make([]string, 0, len(lists))
	evLists := make([]webhooks.EventList, 0, len(lists))
	for _, l := range lists {
		uuids = append(uuids, l.UUID)
		evLists = append(evLists, webhooks.EventList{UUID: l.UUID, Name: l.Name})
	}
	sort.Strings(uuids)

	return webhooks.Event{
		Event:          webhooks.EventSubscriberConfirmed,
		Test:           test,
		IdempotencyKey: fmt.Sprintf("%s:%s", sub.UUID, strings.Join(uuids, ",")),
		ConfirmedAt:    at,
		Subscriber: webhooks.Subscriber{
			UUID:  sub.UUID,
			Email: sub.Email,
			Name:  sub.Name,
		},
		Lists: evLists,
	}
}

// GetListWebhookSecret reveals a list's webhook signing secret so admins
// can verify or copy it into the receiving app. Requires list manage
// permission; the secret is never included in normal list responses.
func (a *App) GetListWebhookSecret(c echo.Context) error {
	user := auth.GetUser(c)
	id := getID(c)
	if err := user.HasListPerm(auth.PermTypeManage, id); err != nil {
		return err
	}

	l, err := a.core.GetList(id, "")
	if err != nil {
		return err
	}

	hooks, err := a.core.GetListWebhooks([]string{l.UUID})
	if err != nil || len(hooks) == 0 || hooks[0].Secret == "" {
		return echo.NewHTTPError(http.StatusNotFound, "no webhook secret configured for this list")
	}

	return c.JSON(http.StatusOK, okResp{map[string]any{"webhook_secret": hooks[0].Secret}})
}

// TestListWebhook sends a signed test event to a list's webhook target so
// admins can verify the integration on save. Synchronous: the result is
// reported back to the caller.
func (a *App) TestListWebhook(c echo.Context) error {
	user := auth.GetUser(c)
	id := getID(c)
	if err := user.HasListPerm(auth.PermTypeManage, id); err != nil {
		return err
	}

	l, err := a.core.GetList(id, "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(l.WebhookURL) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "no webhook URL configured for this list")
	}

	hooks, err := a.core.GetListWebhooks([]string{l.UUID})
	if err != nil || len(hooks) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no webhook URL configured for this list")
	}
	h := hooks[0]

	ev := webhooks.Event{
		Event:          webhooks.EventSubscriberConfirmed,
		Test:           true,
		IdempotencyKey: fmt.Sprintf("test-%d", time.Now().UnixNano()),
		ConfirmedAt:    time.Now().UTC().Format(time.RFC3339),
		Subscriber:     webhooks.Subscriber{UUID: "00000000-0000-0000-0000-000000000000", Email: "webhook-test@example.com", Name: "Webhook Test"},
		Lists:          []webhooks.EventList{{UUID: l.UUID, Name: l.Name}},
	}

	d := webhooks.NewDispatcher(a.log)
	attempt, err := d.SendDetailed(webhooks.Target{ListID: l.ID, ListUUID: l.UUID, ListName: l.Name, URL: h.URL, Secret: h.Secret}, ev)
	out := map[string]any{"url": h.URL, "sent": attempt}
	if err != nil {
		out["delivered"] = false
		return c.JSON(http.StatusBadGateway, out)
	}

	out["delivered"] = true
	return c.JSON(http.StatusOK, okResp{out})
}

// validateListWebhookURL ensures an optional list webhook URL is sane.
func validateListWebhookURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return webhooks.ValidateURL(raw)
}
