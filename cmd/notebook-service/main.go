package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/vasantbala/notebook-service/internal/api"
	"github.com/vasantbala/notebook-service/internal/service"
)

func main() {

	fmt.Println("notebook-service starting up...")
	svc := service.NewInMemNotebookService()
	h := &api.Handlers{Notebooks: svc}
	r := api.NewRouter(h)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
