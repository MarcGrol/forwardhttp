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

func (w Warehouse) Put(c context.Context, req httpclient.Request, resp *httpclient.Response, err error, stats Stats) error {
	fs := &forwardStatsRecord{
		Request:  req,
		Response: resp,
		ErrorMsg: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		Stats:     stats,
		Completed: stats.RetryCount == stats.MaxRetryCount,
	}

	putErr := w.store.Put(c, "forwardStatsRecord", req.UID, fs)
	if putErr != nil {
		log.Printf("Error storing task-status: %s", err)
		return fmt.Errorf("Error storing task-status: %s", err)
	}
	return nil
}
