package vo

type DocumentParseRouteMessage struct {
	DocumentId int64 `json:"documentId"`
	TaskId     int64 `json:"taskId"`
}

type DocumentIndexBuildMessage struct {
	DocumentId int64 `json:"documentId"`
	TaskId     int64 `json:"taskId"`
	PlanId     int64 `json:"planId"`
}
