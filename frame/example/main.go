package main

import (
	"log"

	"moarchan/frame"
	"github.com/joncody/wsrooms"
)

var app *frame.App

func testHandler(c *wsrooms.Conn, msg *wsrooms.Message, matches []string) {
	log.Println(matches)
	app.Render(c, msg, "index-added", []string{"index"}, nil)
}

func main() {
	app = frame.NewApp("./config.json")
	app.AddRoute("^/test/(.*)$", testHandler)
	app.Start()
}
