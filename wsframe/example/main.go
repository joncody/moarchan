package main

import (
	"log"

	"moarchan/wsframe"
	"github.com/joncody/wsrooms"
)

var app *wsframe.App

func testHandler(c *wsrooms.Conn, msg *wsrooms.Message, matches []string) {
	log.Println(matches)
	app.Render(c, msg, "index-added", []string{"index"}, nil)
}

func main() {
	app = wsframe.NewApp("./config.json")
	app.AddRoute("^/test/(.*)$", testHandler)
	app.Start()
}
