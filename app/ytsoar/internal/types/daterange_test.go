package types_test

import (
	"testing"
	"time"

	"github.com/yuudev14/ytsoar/internal/domain/apperr"
	"github.com/yuudev14/ytsoar/internal/types"
)

func ptr(s string) *string { return &s }

func TestResolveDefaultsToTrailingFortnight(t *testing.T) {
	rng, err := types.DateRange{}.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got := rng.Span(); got < 13*24*time.Hour || got > 15*24*time.Hour {
		t.Fatalf("default span = %v, want ~14 days", got)
	}
	if rng.Bucket != types.BucketDay {
		t.Fatalf("default bucket = %q, want %q", rng.Bucket, types.BucketDay)
	}
}

func TestResolveDerivesBucketFromSpan(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want string
	}{
		{"two weeks is daily", "2026-07-01T00:00:00Z", "2026-07-15T00:00:00Z", types.BucketDay},
		{"six months is weekly", "2026-01-01T00:00:00Z", "2026-07-01T00:00:00Z", types.BucketWeek},
		{"five years is monthly", "2021-01-01T00:00:00Z", "2026-01-01T00:00:00Z", types.BucketMonth},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rng, err := types.DateRange{From: ptr(tc.from), To: ptr(tc.to)}.Resolve()
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if rng.Bucket != tc.want {
				t.Fatalf("bucket = %q, want %q", rng.Bucket, tc.want)
			}
		})
	}
}

// An explicit bucket must win over the derived one, otherwise a weekly view of a
// 30-day range is unaskable.
func TestResolveExplicitBucketBeatsDerived(t *testing.T) {
	rng, err := types.DateRange{
		From:   ptr("2026-06-01T00:00:00Z"),
		To:     ptr("2026-07-01T00:00:00Z"),
		Bucket: ptr(types.BucketWeek),
	}.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if rng.Bucket != types.BucketWeek {
		t.Fatalf("bucket = %q, want %q", rng.Bucket, types.BucketWeek)
	}
}

func TestResolveRejectsTooManyPoints(t *testing.T) {
	_, err := types.DateRange{
		From:   ptr("2016-01-01T00:00:00Z"),
		To:     ptr("2026-01-01T00:00:00Z"),
		Bucket: ptr(types.BucketDay),
	}.Resolve()

	if kind, _ := apperr.KindOf(err); kind != apperr.Invalid {
		t.Fatalf("kind = %v, want Invalid (err=%v)", kind, err)
	}
}

// The same decade is fine once the bucket is wide enough - the guard is on the
// point count, not on how far back a caller may look.
func TestResolveAllowsLongRangeWithWideBucket(t *testing.T) {
	rng, err := types.DateRange{
		From:   ptr("2016-01-01T00:00:00Z"),
		To:     ptr("2026-01-01T00:00:00Z"),
		Bucket: ptr(types.BucketMonth),
	}.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if rng.Bucket != types.BucketMonth {
		t.Fatalf("bucket = %q, want %q", rng.Bucket, types.BucketMonth)
	}
}

func TestResolveRejectsInvertedRange(t *testing.T) {
	_, err := types.DateRange{
		From: ptr("2026-07-20T00:00:00Z"),
		To:   ptr("2026-07-01T00:00:00Z"),
	}.Resolve()

	if kind, _ := apperr.KindOf(err); kind != apperr.Invalid {
		t.Fatalf("kind = %v, want Invalid (err=%v)", kind, err)
	}
}

func TestResolveRejectsMalformedBound(t *testing.T) {
	_, err := types.DateRange{From: ptr("2026-07-20")}.Resolve()
	if kind, _ := apperr.KindOf(err); kind != apperr.Invalid {
		t.Fatalf("kind = %v, want Invalid (err=%v)", kind, err)
	}
}

// Previous is what every rendered delta is measured against, so it must abut the
// selected window exactly and share its length.
func TestPreviousWindowAbutsAndMatchesLength(t *testing.T) {
	rng, err := types.DateRange{
		From: ptr("2026-07-15T00:00:00Z"),
		To:   ptr("2026-07-29T00:00:00Z"),
	}.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	prevFrom, prevTo := rng.Previous()
	if !prevTo.Equal(rng.From) {
		t.Fatalf("previous window ends at %v, want %v", prevTo, rng.From)
	}
	if got := prevTo.Sub(prevFrom); got != rng.Span() {
		t.Fatalf("previous span = %v, want %v", got, rng.Span())
	}
}

func TestParseInstantNilPassthrough(t *testing.T) {
	got, err := types.ParseInstant(nil)
	if err != nil || got != nil {
		t.Fatalf("ParseInstant(nil) = %v, %v; want nil, nil", got, err)
	}

	empty := ""
	got, err = types.ParseInstant(&empty)
	if err != nil || got != nil {
		t.Fatalf(`ParseInstant("") = %v, %v; want nil, nil`, got, err)
	}
}
