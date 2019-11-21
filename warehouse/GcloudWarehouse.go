package warehouse

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MarcGrol/forwardhttp/httpclient"
	"github.com/MarcGrol/forwardhttp/store"
)

type Warehouse struct {
	store store.DataStorer
}

func New(store store.DataStorer) Warehouser {
	return &Warehouse{
		store: store,
	}
}

type forwardStatsRecord struct {
	Timestamp time.Time
	Request   httpclient.Request
	Response  *httpclient.Response
	ErrorMsg  string
	Stats     Stats
	Completed bool
}

func (w Warehouse) Put(c context.Context, summary ForwardSummary) error {
	fs := &forwardStatsRecord{
		Timestamp: time.Now(),
		Request:   summary.HttpRequest,
		Response:  summary.HttpResponse,
		ErrorMsg: func() string {
			if summary.Error != nil {
				return summary.Error.Error()
			}
			return ""
		}(),
		Stats:     summary.Stats,
		Completed: summary.Stats.IsLastAttempt(),
	}

	putErr := w.store.Put(c, "ForwardSummary", summary.HttpRequest.TaskUID, fs)
	if putErr != nil {
		log.Printf("Error storing task-status: %s", putErr)
		return fmt.Errorf("Error storing task-status: %s", putErr)
	}
	return nil
}
