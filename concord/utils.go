package concord

import (
	"time"

	tmtime "github.com/tendermint/tendermint/types/time"
)

const (
	RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
)

// CanonicalTime can be used to stringify time in a canonical way.
//
//	TODO: rework
func CanonicalTime(t time.Time) string {
	// Note that sending time over amino resets it to
	// local time, we need to force UTC here, so the
	// signatures match
	return tmtime.Canonical(t).Format(RFC3339Nano)
}
