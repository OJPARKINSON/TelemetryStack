package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ojparkinson/IRacing-Display/cloud/internal/handlers"
)

func testHandler(w http.ResponseWriter, r *http.Request) {
	country := os.Getenv("CLOUDFLARE_COUNTRY_A2")
	location := os.Getenv("CLOUDFLARE_LOCATION")
	region := os.Getenv("CLOUDFLARE_REGION")

	fmt.Fprintf(w, "Hi, I'm a container running in %s, %s, which is part of %s ", location, country, region)
}

func main() {
	var port string
	var exists bool
	if port, exists = os.LookupEnv("PORT"); !exists {
		port = "8080"
	}

	c := make(chan os.Signal, 10)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	terminate := false
	go func() {
		for range c {
			if terminate {
				os.Exit(0)
				continue
			}

			terminate = true
			go func() {
				time.Sleep(time.Minute)
				os.Exit(0)
			}()
		}
	}()

	mux := http.NewServeMux()

	h := &handlers.Handler{}

	mux.HandleFunc("/test", testHandler)
	mux.HandleFunc("/api/geojson", h.GetGeojson)
	mux.HandleFunc("/api/sessions", h.GetSessions)
	mux.HandleFunc("/api/session/{sessionId}", h.GetSession)

	server := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: mux,
	}

	fmt.Println("server started on 0.0.0.0:" + port)
	log.Fatal(server.ListenAndServe())
}
