package enum

type RouteStatus = string

const (
	RouteStatusSuccess       = "SUCCESS"
	RouteStatusLowConfidence = "LOW_CONFIDENCE"
	RouteStatusFailed        = "FAILED"
)
