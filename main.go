package main

import (
	"fmt"
	"net/http"

	handler "transit-api/handler"

	"github.com/jtclarkjr/router-go"
	"github.com/jtclarkjr/router-go/middleware"
)

func main() {
	router := router.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.RateLimiter)
	router.Use(middleware.Throttle((100)))

	router.Use(middleware.EnvVarChecker("RAPIDAPI_KEY", "RAPIDAPI_TRANSPORT_HOST", "RAPIDAPI_TRANSIT_HOST"))

	router.Get("/transit", handler.Transit())
	router.Get("/autocomplete", handler.Autocomplete)

	fmt.Println("Starting server on :8080")
	http.ListenAndServe(":8080", router)
}
