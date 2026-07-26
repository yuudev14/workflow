package repository

import (
	"github.com/yuudev14/ytsoar/internal/application/alerts"
	"github.com/yuudev14/ytsoar/internal/application/incidents"
)

// The alert repository serves two consumers: its own module, and the incidents
// service, which writes onto an alert's timeline when linking. cmd/api/main.go
// passes the same instance to both, so a signature drift on either port would
// otherwise only surface at composition.
var (
	_ alerts.AlertRepository       = (*AlertRepositoryImpl)(nil)
	_ incidents.AlertTimeline      = (*AlertRepositoryImpl)(nil)
	_ incidents.IncidentRepository = (*IncidentRepositoryImpl)(nil)
)
