package auth

import (
	"context"

	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/types"
)

// ListAuditLogs is read-only by design: audit rows are never edited or deleted
// through the API, or the trail could be doctored by whoever it incriminates.
func (s *Service) ListAuditLogs(ctx context.Context, offset, limit int, filter AuditFilter) (types.Entries[domain.AuditLog], error) {
	logs, err := s.audit.List(ctx, offset, limit, filter)
	if err != nil {
		return types.Entries[domain.AuditLog]{}, err
	}

	total, err := s.audit.Count(ctx, filter)
	if err != nil {
		return types.Entries[domain.AuditLog]{}, err
	}

	return types.Entries[domain.AuditLog]{Entries: logs, Total: total}, nil
}
