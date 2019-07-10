package web

import "time"

type TaskStatus struct {
	UID          string
	Timestamp    time.Time
	Success      bool
	Method       string
	URL          string
	RequestBody  string `datastore:",noindex"`
	ResponseBody string `datastore:",noindex"`
}
