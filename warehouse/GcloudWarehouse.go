package warehouse

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MarcGrol/forwardhttp/datastore"
	"github.com/MarcGrol/forwardhttp/httpclient"
)

type Warehouse struct {
	store datastore.DataStorer
}

func New(store datastore.DataStorer) Warehouser {
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
		Request:  summary.HttpRequest,
		Response: summary.HttpResponse,
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
