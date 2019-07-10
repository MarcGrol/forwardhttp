package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	c := context.Background()

	var router = mux.NewRouter()

	q, err := NewQueue(c)
	if err != nil {
		log.Fatalf("Error creating queue: %s", err)
	}

	f := &ForwarderService{}
	f.HTTPHandlerWithRouter(router)

	r := &ReceiverService{
		queue: q,
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
