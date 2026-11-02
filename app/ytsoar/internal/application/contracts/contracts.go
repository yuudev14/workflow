package contracts

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

//go:generate mockgen -destination=mocks/audit_writer_mock.go -package=mocks . AuditWriter

// AuditWriter writes one audit log row. Each service that audits takes this one port.
type AuditWriter interface {
	Insert(ctx context.Context, entry domain.AuditEntry) error
}

//go:generate mockgen -destination=mocks/run_enqueuer_mock.go -package=mocks . RunEnqueuer

// RunEnqueuer queues a playbook run for a worker. It joins the transaction in ctx, so the
// job and the history rows commit or roll back together. A run is never queued without its
// history.
type RunEnqueuer interface {
	EnqueueRun(ctx context.Context, playbookHistoryID uuid.UUID) error
}

//go:generate mockgen -destination=mocks/status_broadcaster_mock.go -package=mocks . StatusBroadcaster

// StatusBroadcaster sends playbook and task status updates to the connected clients.
type StatusBroadcaster interface {
	Broadcast(data any)
}

//go:generate mockgen -destination=mocks/module_event_publisher_mock.go -package=mocks . ModuleEventPublisher

// ModuleEventPublisher announces an entity event (alert.created, incident.updated, ...),
// so consumers can react without the alerts and incidents services knowing them.
//
// Publish inside the transaction of the change. It joins the transaction in ctx, so a
// rollback also drops the event, and a commit never loses it.
type ModuleEventPublisher interface {
	Publish(ctx context.Context, module, event string, recordID uuid.UUID) error
}

//go:generate mockgen -destination=mocks/tx_manager_mock.go -package=mocks . TxManager

// TxManager runs fn in a database transaction carried in the context, so repositories join
// it and services never touch the driver.
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

//go:generate mockgen -destination=mocks/user_directory_mock.go -package=mocks . UserDirectory

// UserDirectory turns user ids into usernames for timeline events. Without it, an
// assignment event reads "assignee changed from 3f2a… to 9b1c…", and who got the alert is
// the whole point of the event.
type UserDirectory interface {
	UsernamesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

//go:generate mockgen -destination=mocks/secret_box_mock.go -package=mocks . SecretBox

// SecretBox encrypts config secrets at rest. The key lives outside the database.
type SecretBox interface {
	Seal(plain string) (string, error)
	Open(sealed string) (string, error)
}

const maskedSecret = "********"

// SealValues seals each top-level string value of a config. It skips the mask, so
// KeepMasked can put the stored value back.
func SealValues(box SecretBox, raw json.RawMessage) (json.RawMessage, error) {
	return mapValues(raw, box.Seal)
}

func OpenValues(box SecretBox, raw json.RawMessage) (json.RawMessage, error) {
	return mapValues(raw, box.Open)
}

// MaskedView opens the values best effort and hides the secret keys, so a row that does not
// open still shows and the browser never gets a live credential.
func MaskedView(box SecretBox, raw json.RawMessage, secretKeys []string) json.RawMessage {
	plain, err := mapValues(raw, func(v string) (string, error) {
		if p, err := box.Open(v); err == nil {
			return p, nil
		}
		return v, nil
	})
	if err != nil {
		return json.RawMessage(`{}`)
	}
	var m map[string]any
	_ = json.Unmarshal(plain, &m)
	for _, key := range secretKeys {
		if v, ok := m[key].(string); ok && v != "" {
			m[key] = maskedSecret
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return out
}

// KeepMasked puts the stored value back for each value that is still the mask, so a round
// trip through the masked view cannot wipe a secret. On a parse problem it returns incoming.
func KeepMasked(stored, incoming json.RawMessage) json.RawMessage {
	var next, prev map[string]any
	if err := json.Unmarshal(incoming, &next); err != nil {
		return incoming
	}
	_ = json.Unmarshal(stored, &prev)
	for key, val := range next {
		if val == maskedSecret {
			next[key] = prev[key]
		}
	}
	merged, err := json.Marshal(next)
	if err != nil {
		return incoming
	}
	return merged
}

func mapValues(raw json.RawMessage, fn func(string) (string, error)) (json.RawMessage, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		return nil, apperr.New(apperr.Invalid, "config must be a JSON object")
	}
	for key, val := range m {
		v, ok := val.(string)
		if !ok || v == "" || v == maskedSecret {
			continue
		}
		out, err := fn(v)
		if err != nil {
			return nil, apperr.Wrap(apperr.Invalid, "config cannot be decrypted, enter it again", err)
		}
		m[key] = out
	}
	return json.Marshal(m)
}
