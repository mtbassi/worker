package redis

// Key patterns for Redis keys.
const (
	KeyPatternJourneyState    = "journey:%s:%s:state"
	KeyPatternJourneyRepiques = "journey:%s:%s:repiques"
	// KeyPatternMessageLock is used for idempotency - prevents duplicate message sends.
	// Format: journey:{journey_id}:{customer_number}:lock:{rule_name}:{attempt_number}
	KeyPatternMessageLock = "journey:%s:%s:lock:%s:%d"
)
