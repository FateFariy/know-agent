package vo

type RouteStatus = string

const (
	RouteStatusSuccess       = "SUCCESS"
	RouteStatusLowConfidence = "LOW_CONFIDENCE"
	RouteStatusFailed        = "FAILED"
)

func RouteStatusCode(name string) int {
	switch name {
	case RouteStatusSuccess:
		return 1
	case RouteStatusLowConfidence:
		return 2
	default:
		return 3
	}
}
