package main

import (
	"fmt"
	"net/http"

	controllers "transit-api/controllers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	loadEnv()
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Get("/transit", controllers.Transit())
	router.Get("/autocomplete", controllers.Autocomplete)

	fmt.Println("Starting server on :3000")
	http.ListenAndServe(":3000", router)
}

func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Printf("Could not load: %v", err)
	}
}
