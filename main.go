package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"product-service/db"
	"product-service/handlers"
	appmw "product-service/middleware"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on system env vars")
	}

	if err := db.Connect(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := db.InitSchema(); err != nil {
		log.Fatalf("failed to init schema: %v", err)
	}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(10 * time.Second))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})
	r.Use(c.Handler)

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/products", func(r chi.Router) {
		r.Get("/", handlers.ListProducts)
		r.Get("/{id}", handlers.GetProduct)

		r.Group(func(r chi.Router) {
			r.Use(appmw.RequireAuth)
			r.Post("/", handlers.AddProduct)
			r.Delete("/{id}", handlers.DeleteProduct)
			r.Patch("/{id}/stock", handlers.AdjustStock)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}

	log.Printf("product-service running on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}