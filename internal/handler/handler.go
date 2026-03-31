package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/suleiman-oss/dogs-server/internal/store"
)

// ── Response helpers ──────────────────────────────────────────────────────────

type envelope map[string]any

func writeJSON(w http.ResponseWriter, status int, data envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

func errorJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, envelope{"status": "error", "message": msg})
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	store *store.Store
}

func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

// Register wires all routes onto the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/dogs", h.collection)
	mux.HandleFunc("/api/dogs/", h.resource) // trailing slash catches sub-paths
}

// ── /api/dogs ─────────────────────────────────────────────────────────────────

func (h *Handler) collection(w http.ResponseWriter, r *http.Request) {
	addCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listAll(w, r)
	case http.MethodPost:
		h.createBreed(w, r)
	default:
		errorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GET /api/dogs
func (h *Handler) listAll(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, envelope{
		"status": "success",
		"data":   h.store.All(),
	})
}

// POST /api/dogs
func (h *Handler) createBreed(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Breed     string   `json:"breed"`
		SubBreeds []string `json:"subBreeds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	breed := normaliseBreed(body.Breed)
	if breed == "" {
		errorJSON(w, http.StatusBadRequest, "breed is required")
		return
	}
	if err := h.store.Create(breed, body.SubBreeds); err != nil {
		errorJSON(w, http.StatusConflict, err.Error())
		return
	}
	subs, _ := h.store.Get(breed)
	writeJSON(w, http.StatusCreated, envelope{
		"status":    "success",
		"message":   "breed \"" + breed + "\" created",
		"breed":     breed,
		"subBreeds": subs,
	})
}

// ── /api/dogs/:breed  and  /api/dogs/:breed/:sub ──────────────────────────────

func (h *Handler) resource(w http.ResponseWriter, r *http.Request) {
	addCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Strip leading "/api/dogs/"
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/dogs/"), "/")

	breed := normaliseBreed(parts[0])
	if breed == "" {
		errorJSON(w, http.StatusBadRequest, "breed is required")
		return
	}

	// Two-segment: /api/dogs/:breed/:subbreed
	if len(parts) == 2 && parts[1] != "" {
		sub := strings.ToLower(strings.TrimSpace(parts[1]))
		if r.Method == http.MethodDelete {
			h.deleteSubBreed(w, r, breed, sub)
		} else {
			errorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	// One-segment: /api/dogs/:breed
	switch r.Method {
	case http.MethodGet:
		h.getBreed(w, r, breed)
	case http.MethodPut:
		h.replaceBreed(w, r, breed)
	case http.MethodPatch:
		h.patchBreed(w, r, breed)
	case http.MethodDelete:
		h.deleteBreed(w, r, breed)
	default:
		errorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// GET /api/dogs/:breed
func (h *Handler) getBreed(w http.ResponseWriter, _ *http.Request, breed string) {
	subs, err := h.store.Get(breed)
	if err != nil {
		errorJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{
		"status":    "success",
		"breed":     breed,
		"subBreeds": subs,
	})
}

// PUT /api/dogs/:breed — replace sub-breeds list
func (h *Handler) replaceBreed(w http.ResponseWriter, r *http.Request, breed string) {
	var body struct {
		SubBreeds []string `json:"subBreeds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.store.Replace(breed, body.SubBreeds); err != nil {
		errorJSON(w, http.StatusNotFound, err.Error())
		return
	}
	subs, _ := h.store.Get(breed)
	writeJSON(w, http.StatusOK, envelope{
		"status":    "success",
		"message":   "breed \"" + breed + "\" updated",
		"breed":     breed,
		"subBreeds": subs,
	})
}

// PATCH /api/dogs/:breed — add sub-breeds
func (h *Handler) patchBreed(w http.ResponseWriter, r *http.Request, breed string) {
	var body struct {
		SubBreeds []string `json:"subBreeds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := h.store.AddSubs(breed, body.SubBreeds); err != nil {
		errorJSON(w, http.StatusNotFound, err.Error())
		return
	}
	subs, _ := h.store.Get(breed)
	writeJSON(w, http.StatusOK, envelope{
		"status":    "success",
		"message":   "sub-breeds added to \"" + breed + "\"",
		"breed":     breed,
		"subBreeds": subs,
	})
}

// DELETE /api/dogs/:breed
func (h *Handler) deleteBreed(w http.ResponseWriter, _ *http.Request, breed string) {
	if err := h.store.DeleteBreed(breed); err != nil {
		errorJSON(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{
		"status":  "success",
		"message": "breed \"" + breed + "\" deleted",
	})
}

// DELETE /api/dogs/:breed/:subbreed
func (h *Handler) deleteSubBreed(w http.ResponseWriter, _ *http.Request, breed, sub string) {
	if err := h.store.DeleteSub(breed, sub); err != nil {
		errorJSON(w, http.StatusNotFound, err.Error())
		return
	}
	subs, _ := h.store.Get(breed)
	writeJSON(w, http.StatusOK, envelope{
		"status":    "success",
		"message":   "sub-breed \"" + sub + "\" removed from \"" + breed + "\"",
		"breed":     breed,
		"subBreeds": subs,
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func normaliseBreed(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
}

func addCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}
