package main

import (
	"fmt"
	"net/http"

	controllers "transit-api/controllers"

	"github.com/jtclarkjr/router-go"
	"github.com/jtclarkjr/router-go/middleware"
)

func main() {
	// loadEnv()
	router := router.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.RateLimiter)
	router.Use(middleware.Throttle((100)))

	router.Get("/transit", controllers.Transit())
	router.Get("/autocomplete", controllers.Autocomplete)

	fmt.Println("Starting server on :8080")
	http.ListenAndServe(":8080", router)
}

// loadEnv loads environment variables from a .env file for local development.
// func loadEnv() {
// 	err := godotenv.Load(".env")
// 	if err != nil {
// 		fmt.Printf("Could not load: %v", err)
// 	}
// }
