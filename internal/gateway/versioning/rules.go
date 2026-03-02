package versioning

import "time"

// DeprecationRule describes a version's deprecation status.
type DeprecationRule struct {
	Version string
	Sunset  time.Time // when it will be removed
	Message string    // human-readable message
}

func (d DeprecationRule) IsDeprecated(now time.Time) bool {
	return !d.Sunset.IsZero() && now.After(d.Sunset)
}
