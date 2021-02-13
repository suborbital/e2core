package vlog

import "time"

type structuredLog struct {
	LogMessage string      `json:"log_message"`
	Timestamp  time.Time   `json:"timestamp"`
	Level      int         `json:"level"`
	AppMeta    interface{} `json:"app,omitempty"`
	ScopeMeta  interface{} `json:"scope,omitempty"`
}
