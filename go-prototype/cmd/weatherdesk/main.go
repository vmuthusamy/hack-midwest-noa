package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/vmuthusamy/hack-midwest-noa/go-prototype/internal/weatherdesk"
)

func main() {
	root := os.Getenv("WEATHER_DESK_ROOT")
	if root == "" {
		root, _ = filepath.Abs("..")
	}
	agentAddress := os.Getenv("WEATHER_DESK_AGENT_ADDR")
	if agentAddress == "" {
		agentAddress = "127.0.0.1:8089"
	}
	app, err := weatherdesk.New(root, agentAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()
	address := os.Getenv("WEATHER_DESK_HTTP_ADDR")
	if address == "" {
		address = "127.0.0.1:8087"
	}
	server := &http.Server{Addr: address, Handler: app, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 100 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
	}()
	log.Println("Weather Desk Go API:", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
