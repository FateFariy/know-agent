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

func RouteStatusName(code int) string {
	switch code {
	case 1:
		return RouteStatusSuccess
	case 2:
		return RouteStatusLowConfidence
	case 3:
		return RouteStatusFailed
	default:
		return ""
	}
}
