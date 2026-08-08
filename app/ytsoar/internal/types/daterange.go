package types

import (
	"time"

	"github.com/yuudev14/ytsoar/internal/domain/apperr"
)

// RFC3339Binding is gin's validator form of time.RFC3339, so a malformed bound
// is rejected at bind time rather than inside a handler.
const RFC3339Binding = "2006-01-02T15:04:05Z07:00"

const (
	defaultRangeDays = 14

	dayBucketMaxDays  = 62
	weekBucketMaxDays = 366

	BucketDay   = "day"
	BucketWeek  = "week"
	BucketMonth = "month"
	BucketYear  = "year"

	// maxSeriesPoints bounds the response and keeps the chart readable. It is a
	// guard on the bucket choice, not on how far back a caller may look.
	maxSeriesPoints = 400
)

// bucketSpans are nominal widths, used only to size the series before running
// the query. date_trunc does the real bucketing.
var bucketSpans = map[string]time.Duration{
	BucketDay:   24 * time.Hour,
	BucketWeek:  7 * 24 * time.Hour,
	BucketMonth: 28 * 24 * time.Hour,
	BucketYear:  365 * 24 * time.Hour,
}

// DateRange binds as *string for the same reason uuids do: gin's binder cannot
// populate a time.Time from a query parameter. Bounds are RFC3339 instants, so
// the offset settles whose midnight a caller meant.
//
// Bucket is the caller's choice of granularity; omitted, it follows from the
// span. Explicit beats derived because a weekly view of a 30-day range is a
// legitimate thing to ask for.
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

// Previous is the window of equal length immediately before this one — the
// baseline every rendered delta is measured against.
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

// bucketFor keeps the returned series a readable length instead of capping how
// far back a caller may ask: a decade of daily points is unusable, a decade of
// monthly points is a chart.
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

// WindowCount pairs a value for the selected range with the same value for the
// preceding range of equal length.
type WindowCount struct {
	Current  int `json:"current"`
	Previous int `json:"previous"`
}

type WindowRate struct {
	Current  float64 `json:"current"`
	Previous float64 `json:"previous"`
}
