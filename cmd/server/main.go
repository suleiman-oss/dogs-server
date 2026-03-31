package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/suleiman-oss/dogs-server/internal/handler"
	"github.com/suleiman-oss/dogs-server/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Resolve paths relative to the binary's working directory
	dataFile := envOrDefault("DATA_FILE", "data/dogs.json")
	seedFile := envOrDefault("SEED_FILE", "data/seed.json")

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(dataFile), 0755); err != nil {
		log.Fatalf("creating data dir: %v", err)
	}

	s, err := store.New(dataFile, seedFile)
	if err != nil {
		log.Fatalf("initialising store: %v", err)
	}
	log.Printf("store loaded from %s", dataFile)

	mux := http.NewServeMux()

	// API routes
	h := handler.New(s)
	h.Register(mux)

	// Serve static frontend — everything else falls through to index.html
	staticDir := envOrDefault("STATIC_DIR", "frontend/public")
	fs := http.FileServer(http.Dir(staticDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If file exists on disk, serve it; otherwise serve index.html (SPA fallback)
		path := filepath.Join(staticDir, r.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})

	addr := ":" + port
	log.Printf("🐕 Dogs API listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
