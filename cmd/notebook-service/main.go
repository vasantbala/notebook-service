package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/vasantbala/notebook-service/internal/api"
)

func main() {
	fmt.Println("notebook-service starting up...")

	r := api.NewRouter()
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}