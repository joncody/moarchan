// main.go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joncody/wsrooms"
	"moarchan/frame"
)

var app *frame.App

func testHandler(c *wsrooms.Conn, msg *wsrooms.Message, matches []string) {
	log.Printf("Test route matched: %v", matches)
	app.Render(c, msg, "index-added", []string{"index"}, nil)
}

func main() {
	// Initialize app
	app, err := frame.NewApp("./config.json")
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	// Add dynamic routes
	if err := app.AddRoute(`^/test/(.*)$`, testHandler); err != nil {
		log.Fatalf("Failed to add route: %v", err)
	}
	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Start server in a goroutine so we can wait for signals
	go func() {
		if err := app.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
		// Optionally: close channel to signal exit
	}()
	log.Printf("✅ %s started on port %s", app.Name, app.Port)
	if app.SSLPort != "0" {
		log.Printf("🔒 HTTPS enabled on port %s", app.SSLPort)
	}
	// Wait for interrupt signal
	<-sigChan
	log.Println(" Shutting down...")
	// Clean up resources
	if err := app.Close(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}
