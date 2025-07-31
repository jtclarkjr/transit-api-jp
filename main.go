package main

import (
	"fmt"
	"net/http"

	"transit-api/handler"

	"github.com/jtclarkjr/router-go"
	"github.com/jtclarkjr/router-go/middleware"
)

func main() {
	r := router.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.RateLimiter)
	r.Use(middleware.Throttle(100))
	r.Use(middleware.EnvVarChecker("RAPIDAPI_KEY", "RAPIDAPI_TRANSPORT_HOST", "RAPIDAPI_TRANSIT_HOST"))

	r.Get("/transit", handler.Transit())
	r.Get("/autocomplete", handler.Autocomplete)

	fmt.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		return
	}
}
