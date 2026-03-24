package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/MicahParks/keyfunc/v3"

	"github.com/vasantbala/notebook-service/internal/api"
	"github.com/vasantbala/notebook-service/internal/service"
)

func main() {

	fmt.Println("notebook-service starting up...")

	jwks, err := keyfunc.NewDefault([]string{"https://auth.curiousabouttech.com/application/o/rag-anything/jwks/"})
	if err != nil {
		log.Fatalf("failed to fetch JWKS: %v", err)
	}

	svc := service.NewInMemNotebookService()
	h := &api.Handlers{Notebooks: svc}
	r := api.NewRouter(h, jwks)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
