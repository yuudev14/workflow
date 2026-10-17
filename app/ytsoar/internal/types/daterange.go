package types

import (
	"time"

	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

// RFC3339Binding is the gin validator form of time.RFC3339, so a bad bound is rejected at
// bind time, not in a handler.
const RFC3339Binding = "2006-01-02T15:04:05Z07:00"

const (
	defaultRangeDays = 14

	dayBucketMaxDays  = 62
	weekBucketMaxDays = 366

	BucketDay   = "day"
	BucketWeek  = "week"
	BucketMonth = "month"
	BucketYear  = "year"

	// maxSeriesPoints limits the response and keeps the chart readable. It guards the bucket
	// choice, not how far back a caller can look.
	maxSeriesPoints = 400
)

// bucketSpans are rough widths, only to size the series before the query. date_trunc does
// the real bucketing.
var bucketSpans = map[string]time.Duration{
	BucketDay:   24 * time.Hour,
	BucketWeek:  7 * 24 * time.Hour,
	BucketMonth: 28 * 24 * time.Hour,
	BucketYear:  365 * 24 * time.Hour,
}

// DateRange binds as *string, like uuids: gin's binder cannot fill a time.Time from a query
// parameter. Bounds are RFC3339 instants, so the offset says whose midnight is meant.
//
// Bucket is the granularity the caller picks. If omitted, it comes from the span. Explicit
// wins, because a weekly view of a 30-day range is a valid request.
type DateRange struct {
	From   *string `form:"from" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	To     *string `form:"to" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	Bucket *string `form:"bucket" binding:"omitempty,oneof=day week month year"`
}

type ResolvedRange struct {
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	Bucket string    `json:"bucket"`
}

func (r ResolvedRange) Span() time.Duration { return r.To.Sub(r.From) }

// Previous is the window of the same length right before this one. Every delta uses it as
// the baseline.
func (r ResolvedRange) Previous() (time.Time, time.Time) {
	return r.From.Add(-r.Span()), r.From
}

// ParseInstant reads an optional RFC3339 bound. Nil in, nil out.
func ParseInstant(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return nil, apperr.New(apperr.Invalid, "invalid timestamp: expected RFC3339")
	}
	return &t, nil
}

func (d DateRange) Resolve() (ResolvedRange, error) {
	from, err := ParseInstant(d.From)
	if err != nil {
		return ResolvedRange{}, err
	}
	to, err := ParseInstant(d.To)
	if err != nil {
		return ResolvedRange{}, err
	}

	end := time.Now().UTC()
	if to != nil {
		end = *to
	}
	start := end.AddDate(0, 0, -defaultRangeDays)
	if from != nil {
		start = *from
	}

	if start.After(end) {
		return ResolvedRange{}, apperr.New(apperr.Invalid, "from must not be after to")
	}

	bucket := bucketFor(end.Sub(start))
	if d.Bucket != nil && *d.Bucket != "" {
		bucket = *d.Bucket
		if points := end.Sub(start) / bucketSpans[bucket]; points > maxSeriesPoints {
			return ResolvedRange{}, apperr.New(apperr.Invalid,
				"range is too long for this bucket: widen the bucket or shorten the range")
		}
	}

	return ResolvedRange{From: start, To: end, Bucket: bucket}, nil
}

// bucketFor keeps the series at a readable length, instead of limiting how far back a
// caller can ask: a decade of daily points is unusable, a decade of monthly points is a
// chart.
func bucketFor(span time.Duration) string {
	switch {
	case span <= dayBucketMaxDays*24*time.Hour:
		return BucketDay
	case span <= weekBucketMaxDays*24*time.Hour:
		return BucketWeek
	default:
		return BucketMonth
	}
}

// WindowCount is a value for the selected range and the same value for the range before.
type WindowCount struct {
	Current  int `json:"current"`
	Previous int `json:"previous"`
}

type WindowRate struct {
	Current  float64 `json:"current"`
	Previous float64 `json:"previous"`
}
