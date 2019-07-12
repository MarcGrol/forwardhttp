package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/MarcGrol/forwardhttp/lastdelivery"

	"github.com/MarcGrol/forwardhttp/entrypoint"
	"github.com/MarcGrol/forwardhttp/forwarder"
	"github.com/MarcGrol/forwardhttp/httpclient"
	"github.com/MarcGrol/forwardhttp/queue"
	store2 "github.com/MarcGrol/forwardhttp/store"
	"github.com/MarcGrol/forwardhttp/warehouse"
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

	httpClient := httpclient.NewClient()

	warehouse := warehouse.New(store)

	lastdeliverer := lastdelivery.NewLastDelivery()

	forwarder := forwarder.NewService(queue, httpClient, warehouse, lastdeliverer)
	forwarder.RegisterEndPoint(router)

	entrypoint := entrypoint.NewWebService(forwarder)
	entrypoint.RegisterEndpoint(router)

	http.Handle("/", router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
