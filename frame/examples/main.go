package main

import (
	"log"
	"net/http"

	"moarchan/frame"
)

func testHandler(app *frame.App, w http.ResponseWriter, r *http.Request, matches []string) {
	log.Printf("Test route matched: %v", matches)
}

func main() {
	app, err := frame.NewApp()
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Programmatic Route Declarations
	app.Route("/").Template("index").Controller("index")
	app.Route(`^/test/(.*)$`).Template("index-added").Controller("index")

	if err := app.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
