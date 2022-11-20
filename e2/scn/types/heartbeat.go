package types

// HeartbeatRequest is a request to send heartbeat data.
type HeartbeatRequest struct {
	Version   string         `json:"version"`
	Runnables *RunnableStats `json:"runnables"`
}

// RunnableStats are stats about runnables.
type RunnableStats struct {
	TotalCount int `json:"totalCount"`
	IdentCount int `json:"identCount"`
}
