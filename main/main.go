package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	store2 "github.com/MarcGrol/forwardhttp/store"

	"github.com/MarcGrol/forwardhttp/queue"
	"github.com/MarcGrol/forwardhttp/web"
	"github.com/gorilla/mux"
)

func main() {
	c := context.Background()

	var router = mux.NewRouter()

	queue, qcleanup, err := queue.NewQueue(c)
	if err != nil {
		log.Fatalf("Error creating queue: %s", err)
	}
	defer qcleanup()

	store, scleanup, err := store2.NewStore(c)
	if err != nil {
		log.Fatalf("Error creating queue: %s", err)
	}
	defer scleanup()

	f := &web.WorkerService{
		Queue: queue,
		Store: store,
	}
	f.HTTPHandlerWithRouter(router)

	r := &web.CommandHandlerService{
		Queue: queue,
	}
	r.HTTPHandlerWithRouter(router)

	http.Handle("/", router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
