package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNextRuns(t *testing.T) {
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		require.NoError(t, err)
		return v
	}
	times := func(spec, tz, after string, n int) []string {
		got, err := NextRuns(spec, tz, at(after), n)
		require.NoError(t, err)
		out := make([]string, len(got))
		for i, g := range got {
			out[i] = g.Format(time.RFC3339)
		}
		return out
	}

	t.Run("returns the next times of a 5-field cron", func(t *testing.T) {
		require.Equal(t, []string{"2026-11-13T12:15:00Z", "2026-11-13T12:30:00Z"},
			times("*/15 * * * *", "UTC", "2026-11-13T12:00:00Z", 2))
	})

	t.Run("never returns the start time itself", func(t *testing.T) {
		require.Equal(t, []string{"2026-11-14T00:00:00Z"}, times("0 0 * * *", "UTC", "2026-11-13T00:00:00Z", 1))
	})

	t.Run("uses the time zone", func(t *testing.T) {
		require.Equal(t, []string{"2026-11-13T14:00:00Z"}, times("0 9 * * *", "America/New_York", "2026-11-13T12:00:00Z", 1))
	})

	t.Run("runs a time that does not exist in a DST gap at the next valid time", func(t *testing.T) {
		// 2026-03-08 02:30 does not exist in New York: 02:00 jumps to 03:00 (EDT)
		got := times("30 2 * * *", "America/New_York", "2026-03-08T00:00:00Z", 2)
		require.Equal(t, []string{"2026-03-08T07:30:00Z", "2026-03-09T06:30:00Z"}, got)
	})

	t.Run("runs a time that occurs two times in a DST overlap one time", func(t *testing.T) {
		// 2026-11-01 01:30 occurs in EDT (05:30Z) and in EST (06:30Z)
		got := times("30 1 * * *", "America/New_York", "2026-11-01T00:00:00Z", 2)
		require.Equal(t, []string{"2026-11-01T05:30:00Z", "2026-11-02T06:30:00Z"}, got)
	})

	t.Run("refuses a bad cron", func(t *testing.T) {
		for _, spec := range []string{"", "@daily", "@hourly", "* * * *", "0 0 * * * *", "61 * * * *", "CRON_TZ=UTC * * * * *", "a b c d e"} {
			_, err := NextRuns(spec, "UTC", time.Now(), 1)
			require.ErrorIs(t, err, ErrInvalidCron, spec)
		}
	})

	t.Run("refuses an unknown time zone", func(t *testing.T) {
		for _, tz := range []string{"", "Mars/Base", "Local"} {
			_, err := NextRuns("* * * * *", tz, time.Now(), 1)
			require.ErrorIs(t, err, ErrInvalidTimezone, tz)
		}
	})
}
