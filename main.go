package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/naruebaet/bitkubsdk/src/model"
	"github.com/naruebaet/bitkubsdk/src/pkg/color"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gorilla/websocket"
	"github.com/naruebaet/bitkubsdk"
)

type Client struct {
	name   string
	events chan *DashBoard
}

type DashBoard struct {
	result model.TickerResponseResult
}

func main() {
	app := fiber.New()

	app.Use(filesystem.New(filesystem.Config{
		Root: http.Dir("./public"),
	}))

	app.Get("/sse", adaptor.HTTPHandler(handler(dashboardHandler)))
	_ = app.Listen(":8080")
}

func handler(f http.HandlerFunc) http.Handler {
	return f
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	client := &Client{name: r.RemoteAddr, events: make(chan *DashBoard, 10)}
	go updateDashboard(client)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	timeout := time.After(1 * time.Second)
	select {
	case ev := <-client.events:
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		_ = enc.Encode(ev.result)
		_, _ = fmt.Fprintf(w, "data: %v\n\n", buf.String())

		log.Printf("read : %v\n\n", buf.String())

	case <-timeout:
		_, _ = fmt.Fprintf(w, ": nothing to sent\n\n")
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func updateDashboard(client *Client) {

	log.Println("updateDashboard")

	bksdk := bitkubsdk.NewBitkub(os.Getenv("BITKUB_API_KEY"), os.Getenv("BITKUB_API_SECRET"))
	ctx, _ := context.WithTimeout(context.Background(),time.Second*20)
	bksdk.WatchTicker(ctx, func(conn *websocket.Conn) {
		defer conn.Close()
		for {

			var output model.TickerResponseResult

			err := conn.ReadJSON(&output)
			if err != nil {
				log.Println(color.ColorRed, "read : ", err)
				return
			}

			db := &DashBoard{
				result: output,
			}

			client.events <- db
		}
	})
}