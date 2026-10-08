package models

import (
	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

const (
	ListTypePrivate    = "private"
	ListTypePublic     = "public"
	ListOptinSingle    = "single"
	ListOptinDouble    = "double"
	ListStatusActive   = "active"
	ListStatusArchived = "archived"
)

// List represents a mailing list.
type List struct {
	Base

	UUID        string         `db:"uuid" json:"uuid"`
	Name        string         `db:"name" json:"name"`
	Type        string         `db:"type" json:"type"`
	Optin       string         `db:"optin" json:"optin"`
	Status      string         `db:"status" json:"status"`
	Tags        pq.StringArray `db:"tags" json:"tags"`
	Description string         `db:"description" json:"description"`

	// Optional per-list webhook fired on subscriber confirmation.
	// The URL is visible to admins; the secret is write-only and never
	// serialized (see ListWebhook for server-side dispatch).
	WebhookURL    string `db:"webhook_url" json:"webhook_url"`
	WebhookSecret string `db:"-" json:"webhook_secret,omitempty"`

	// Optional custom double opt-in confirmation e-mail template body.
	// Empty means the built-in system template (subscriber-optin) is used.
	// Deprecated: superseded by OptinTemplateID.
	OptinTemplate string `db:"optin_template" json:"optin_template,omitempty"`

	// Optional saved template (Templates UI) used for the double opt-in
	// confirmation e-mail. NULL/0 means the built-in system template is used.
	OptinTemplateID null.Int `db:"optin_template_id" json:"optin_template_id"`

	SubscriberCount  int          `db:"subscriber_count" json:"subscriber_count"`
	SubscriberCounts StringIntMap `db:"subscriber_statuses" json:"subscriber_statuses"`
	SubscriberID     int          `db:"subscriber_id" json:"-"`

	// This is only relevant when querying the lists of a subscriber.
	SubscriptionStatus    string    `db:"subscription_status" json:"subscription_status,omitempty"`
	SubscriptionCreatedAt null.Time `db:"subscription_created_at" json:"subscription_created_at,omitempty"`
	SubscriptionUpdatedAt null.Time `db:"subscription_updated_at" json:"subscription_updated_at,omitempty"`

	// Pseudofield for getting the total number of subscribers
	// in searches and queries.
	Total int `db:"total" json:"-"`
}

// ListWebhook is a per-list webhook target for lifecycle integrations.
// Server-side only: it carries the secret, so it must never be serialized
// into API responses.
type ListWebhook struct {
	ListID   int    `db:"id" json:"-"`
	ListUUID string `db:"uuid" json:"-"`
	URL      string `db:"webhook_url" json:"-"`
	Secret   string `db:"webhook_secret" json:"-"`
}
