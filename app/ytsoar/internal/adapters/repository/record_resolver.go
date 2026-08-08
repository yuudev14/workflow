package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// RecordResolverImpl hydrates the module rows a playbook run acts on.
//
// It reads through GetByID, not GetDetail: the plain row carries `payload`, which
// is what templates read, while the detail aggregates pull in timeline, notes and
// linked entities that nothing templates off and that would bloat every node's
// gRPC call.
type RecordResolverImpl struct {
	logger    logger.Logger
	alerts    *AlertRepositoryImpl
	incidents *IncidentRepositoryImpl
}

func NewRecordResolverImpl(log logger.Logger, alerts *AlertRepositoryImpl, incidents *IncidentRepositoryImpl) *RecordResolverImpl {
	return &RecordResolverImpl{logger: log, alerts: alerts, incidents: incidents}
}

func (r *RecordResolverImpl) Resolve(ctx context.Context, moduleType string, ids []uuid.UUID) ([]map[string]any, error) {
	records := make([]map[string]any, 0, len(ids))

	for _, id := range ids {
		var record any
		var err error

		switch moduleType {
		case domain.ModuleEventAlert:
			record, err = r.alerts.GetByID(ctx, id)
		case domain.ModuleEventIncident:
			record, err = r.incidents.GetByID(ctx, id)
		default:
			return nil, apperr.New(apperr.Invalid, "unknown module type: "+moduleType)
		}
		if err != nil {
			return nil, err
		}

		// Round-tripped through JSON so the template engine sees the wire field
		// names (the json tags), not the Go field names.
		encoded, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}
		var decoded map[string]any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			return nil, err
		}
		records = append(records, decoded)
	}

	return records, nil
}
