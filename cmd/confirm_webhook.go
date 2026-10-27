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
// and fans a signed event out asynchronously. It never fails the request;
// all errors are logged.
//
// When env webhook config is present (PATRA_CONFIRM_WEBHOOKS, or the
// PATRA_CONFIRM_WEBHOOK_URL catch-all) it is the single source of truth:
// every confirmed list is matched to its own env target and the per-list
// database webhook columns are ignored. Otherwise, each list with a saved
// webhook URL receives its own list-scoped event.
func (a *App) dispatchConfirmWebhooks(subUUID string, lists []models.List) {
	if len(lists) == 0 {
		return
	}

	sub, err := a.core.GetSubscriber(0, subUUID, "")
	if err != nil {
		a.log.Printf("confirm webhook: cannot load subscriber %s: %v", subUUID, err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	d := webhooks.NewDispatcher(a.log)

	// Env-configured endpoints (preferred): every confirmed list is delivered
	// to its own list-scoped target so each HMAC secret stays per-list.
	if envWebhookConfigured() {
		for _, l := range lists {
			target, ok := resolveEnvWebhook(l)
			if !ok {
				continue
			}
			if err := webhooks.ValidateURL(target.URL); err != nil {
				a.log.Printf("confirm webhook: invalid env target for list %q: %v", l.Name, err)
				continue
			}
			ev := confirmEvent(sub, []models.List{l}, now, false)
			d.Dispatch([]webhooks.Target{target}, ev)
		}
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
}

// envWebhookConfigured reports whether confirm webhooks are configured
// through the environment at all. When true, per-list database targets are
// ignored.
func envWebhookConfigured() bool {
	_, _, configured, _ := webhooks.EnvTarget("", "")
	return configured
}

// resolveEnvWebhook returns the env-configured target for a list, carrying
// the list metadata needed by the dispatcher.
func resolveEnvWebhook(l models.List) (webhooks.Target, bool) {
	u, s, _, ok := webhooks.EnvTarget(l.UUID, l.Name)
	if !ok {
		return webhooks.Target{}, false
	}
	return webhooks.Target{
		ListID:   l.ID,
		ListUUID: l.UUID,
		ListName: l.Name,
		URL:      u,
		Secret:   s,
	}, true
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

// GetListWebhookSecret reveals the webhook signing secret so admins can
// verify or copy it into the receiving app. Requires list manage permission;
// the secret is never included in normal list responses. When webhooks are
// configured through the environment, the matching env secret is returned.
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

	if envWebhookConfigured() {
		if t, ok := resolveEnvWebhook(l); ok && t.Secret != "" {
			return c.JSON(http.StatusOK, okResp{map[string]any{"webhook_secret": t.Secret}})
		}
		return echo.NewHTTPError(http.StatusNotFound, "no webhook secret configured for this list")
	}

	hooks, err := a.core.GetListWebhooks([]string{l.UUID})
	if err != nil || len(hooks) == 0 || hooks[0].Secret == "" {
		return echo.NewHTTPError(http.StatusNotFound, "no webhook secret configured for this list")
	}

	return c.JSON(http.StatusOK, okResp{map[string]any{"webhook_secret": hooks[0].Secret}})
}

// TestListWebhook sends a signed test event to the active webhook target so
// admins can verify the integration on save. The env-configured target for
// the list is used when present, otherwise the list's saved webhook.
// Synchronous: the result is reported back to the caller.
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

	target := webhooks.Target{ListID: l.ID, ListUUID: l.UUID, ListName: l.Name}
	source := "list"
	if envWebhookConfigured() {
		t, ok := resolveEnvWebhook(l)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "no env webhook configured for this list")
		}
		if err := webhooks.ValidateURL(t.URL); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				"invalid env webhook URL: "+err.Error())
		}
		target = t
		source = "env"
	} else {
		if strings.TrimSpace(l.WebhookURL) == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "no webhook URL configured for this list")
		}

		hooks, err := a.core.GetListWebhooks([]string{l.UUID})
		if err != nil || len(hooks) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "no webhook URL configured for this list")
		}
		target.URL = hooks[0].URL
		target.Secret = hooks[0].Secret
	}

	ev := webhooks.Event{
		Event:          webhooks.EventSubscriberConfirmed,
		Test:           true,
		IdempotencyKey: fmt.Sprintf("test-%d", time.Now().UnixNano()),
		ConfirmedAt:    time.Now().UTC().Format(time.RFC3339),
		Subscriber:     webhooks.Subscriber{UUID: "00000000-0000-0000-0000-000000000000", Email: "webhook-test@example.com", Name: "Webhook Test"},
		Lists:          []webhooks.EventList{{UUID: l.UUID, Name: l.Name}},
	}

	d := webhooks.NewDispatcher(a.log)
	attempt, err := d.SendDetailed(target, ev)
	out := map[string]any{"url": target.URL, "source": source, "sent": attempt}
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
