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
	locationID := os.Getenv("LOCATION_ID")

	if locationID == "" {
		log.Fatal("LOCATION_ID env var required")
	}
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		log.Fatal("API_URL env var required")
	}

	daemonStart := time.Now()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	for {
		u := url.URL{Scheme: "ws", Host: apiURL[7:], Path: "/ws", RawQuery: fmt.Sprintf("location_id=%s&uptime=%d", locationID, int(time.Since(daemonStart).Seconds()))}
		if apiURL[:7] == "http://" {
			u.Scheme = "ws"
			u.Host = apiURL[7:]
		} else {
			u.Scheme = "wss"
			u.Host = apiURL[8:]
		}

		log.Printf("connecting to %s", u.String())
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Printf("connection failed: %v — retrying in 10s", err)
			time.Sleep(10 * time.Second)
			continue
		}

		log.Printf("connected — location %s", locationID)

		c.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.SetPongHandler(func(string) error {
			c.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}()

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
