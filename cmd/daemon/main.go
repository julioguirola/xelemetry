package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: daemon <location_id>")
	}
	locationID := os.Args[1]

	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:1323"
	}

	u := url.URL{Scheme: "ws", Host: apiURL[7:], Path: "/ws", RawQuery: fmt.Sprintf("location_id=%s", locationID)}
	if apiURL[:7] == "http://" {
		u.Scheme = "ws"
		u.Host = apiURL[7:]
	} else {
		u.Scheme = "wss"
		u.Host = apiURL[8:]
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	for {
		log.Printf("connecting to %s", u.String())
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Printf("connection failed: %v — retrying in 10s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		log.Printf("connected — location %s", locationID)

		done := make(chan struct{})
		go func() {
			defer close(done)
			for {
				_, _, err := c.ReadMessage()
				if err != nil {
					log.Printf("disconnected: %v", err)
					return
				}
			}
		}()

		select {
		case <-done:
			c.Close()
		case <-sig:
			log.Print("shutting down")
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			c.Close()
			return
		}

		log.Print("reconnecting in 10s")
		time.Sleep(10 * time.Second)
	}
}
