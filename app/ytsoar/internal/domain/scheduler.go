package domain

import (
	"strings"
	"time"
	_ "time/tzdata" // a slim image has no zoneinfo, and a schedule needs the time zone names

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

var (
	ErrInvalidCron     = apperr.New(apperr.Invalid, "cron must have 5 fields: minute hour day-of-month month day-of-week")
	ErrInvalidTimezone = apperr.New(apperr.Invalid, "timezone must be an IANA name, for example Europe/Paris")
)

// Schedule runs one manual playbook on a crontab timetable.
type Schedule struct {
	ID                  uuid.UUID      `json:"id"`
	Name                string         `json:"name"`
	PlaybookID          uuid.UUID      `json:"playbook_id"`
	PlaybookName        string         `json:"playbook_name"`
	PlaybookTriggerType *string        `json:"playbook_trigger_type"`
	PlaybookDeleted     bool           `json:"playbook_deleted"`
	Cron                string         `json:"cron"`
	Timezone            string         `json:"timezone"`
	Enabled             bool           `json:"enabled"`
	Parameters          map[string]any `json:"parameters"`
	NextRunAt           time.Time      `json:"next_run_at"`
	LastRunAt           *time.Time     `json:"last_run_at"`
	CreatedBy           *uuid.UUID     `json:"created_by"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// no cron.Descriptor: @daily and the other short forms are refused
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// NextRuns returns the next n times after `after`. cron is the plain 5-field form. The
// library moves a time that does not exist in the time zone (DST gap) to the next valid
// time, and runs a time that occurs two times (DST overlap) one time.
func NextRuns(spec, timezone string, after time.Time, n int) ([]time.Time, error) {
	if len(strings.Fields(spec)) != 5 {
		return nil, ErrInvalidCron
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil || timezone == "" || timezone == "Local" {
		return nil, ErrInvalidTimezone
	}
	schedule, err := cronParser.Parse(spec)
	if err != nil {
		return nil, apperr.Wrap(apperr.Invalid, "cron is not valid: "+err.Error(), ErrInvalidCron)
	}
	// The library skips a day in a DST gap and runs twice in an overlap. So it counts wall
	// clock times in UTC, and time.Date maps each one to one real time in the zone.
	wall := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
	}
	times := make([]time.Time, 0, n)
	for cursor, last := wall(after.In(loc)), after; len(times) < n; {
		next := schedule.Next(cursor)
		if next.IsZero() { // no match in 5 years, for example 0 0 31 2 *
			break
		}
		cursor = next
		at := time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), next.Minute(), 0, 0, loc)
		if got := at.In(loc); got.Hour() != next.Hour() || got.Minute() != next.Minute() {
			at = at.Add(time.Hour) // DST gap: run when the clock jumps over the missing time
		}
		if at.After(last) {
			times = append(times, at.UTC())
			last = at
		}
	}
	return times, nil
}
