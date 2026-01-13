package redis

// Key patterns for Redis keys.
const (
	KeyPatternJourneyState    = "journey:%s:%s:state"
	KeyPatternJourneyRepiques = "journey:%s:%s:repiques"
)
