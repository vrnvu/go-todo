package todos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

type XRequestIDHeader string

const XRequestIDHeaderKey XRequestIDHeader = "X-Request-ID"

type Handler struct {
	Slog *slog.Logger
	Mux  *http.ServeMux
	db   *DB
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Mux.ServeHTTP(w, r)
}

type Config struct {
	DBFile             string
	Slog               *slog.Logger
	RequestIDGenerator func() string
}

func FromConfig(c *Config) (*Handler, error) {
	db, err := NewDB(c.DBFile)
	if err != nil {
		return nil, err
	}

	h := &Handler{Slog: c.Slog, Mux: http.NewServeMux(), db: db}

	h.Mux.HandleFunc("GET /health", health)
	h.Mux.HandleFunc("GET /todos", withBaseMiddleware(c.Slog, c.RequestIDGenerator, h.getAll))
	h.Mux.HandleFunc("GET /todos/{id}", withBaseMiddleware(c.Slog, c.RequestIDGenerator, h.get))
	h.Mux.HandleFunc("PUT /todos/{id}", withBaseMiddleware(c.Slog, c.RequestIDGenerator, h.insert))
	h.Mux.HandleFunc("DELETE /todos/{id}", withBaseMiddleware(c.Slog, c.RequestIDGenerator, h.delete))
	return h, nil
}

func health(_ http.ResponseWriter, _ *http.Request) {}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid id: `%s`", rawID), http.StatusBadRequest)
		return
	}

	if err := h.db.Delete(r.Context(), id); err != nil {
		h.logError(r, fmt.Sprintf("failed to delete todo with id `%d`", id), err)
		http.Error(w, fmt.Sprintf("failed to delete todo with id `%d`", id), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) insert(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, fmt.Sprintf("invalid content type: `%s`, use `application/json`", contentType), http.StatusBadRequest)
		return
	}

	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid id: `%s`", rawID), http.StatusBadRequest)
		return
	}

	todo := Todo{}
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, "failed to decode todo body", http.StatusBadRequest)
		return
	}

	if id != todo.ID {
		http.Error(w, fmt.Sprintf("id in path `%d` and body `%d` do not match", id, todo.ID), http.StatusBadRequest)
		return
	}

	if err := h.db.Insert(r.Context(), todo); err != nil {
		h.logError(r, "failed to insert todo", err)
		http.Error(w, "failed to insert todo", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) getAll(w http.ResponseWriter, r *http.Request) {
	todos, err := h.db.GetAll(r.Context())
	if err != nil {
		h.logError(r, "failed to get todos", err)
		http.Error(w, "failed to get todos", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, r, todos)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		h.logError(r, fmt.Sprintf("failed to convert id `%s` to uint", rawID), err)
		http.Error(w, fmt.Sprintf("invalid id: `%s`", rawID), http.StatusBadRequest)
		return
	}

	todo, err := h.db.Get(r.Context(), id)
	if err != nil {
		var notFoundErr ErrNotFound
		if errors.As(err, &notFoundErr) {
			http.Error(w, notFoundErr.Error(), http.StatusNotFound)
			return
		}

		h.logError(r, fmt.Sprintf("failed to get todo with id `%d`", id), err)
		http.Error(w, fmt.Sprintf("failed to get todo with id `%d`", id), http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, r, todo)
}

func (h *Handler) writeJSON(w http.ResponseWriter, r *http.Request, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", fromContext(r, XRequestIDHeaderKey))

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logError(r, "failed to encode data", err)
		http.Error(w, "failed to encode data", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) logError(r *http.Request, message string, err error) {
	h.Slog.Error(message, "error", err, string(XRequestIDHeaderKey), fromContext(r, XRequestIDHeaderKey))
}

func fromContext(r *http.Request, key any) string {
	return r.Context().Value(key).(string)
}

func withRequestID(requestIDGenerator func() string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := requestIDGenerator()
		ctx := context.WithValue(r.Context(), XRequestIDHeaderKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func withLoggingMethod(slog *slog.Logger, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("", "method", r.Method, "path", r.URL.Path, "requestID", fromContext(r, XRequestIDHeaderKey))
		next.ServeHTTP(w, r)
	}
}

func withBaseMiddleware(slog *slog.Logger, requestIDGenerator func() string, next http.HandlerFunc) http.HandlerFunc {
	return withRequestID(requestIDGenerator, withLoggingMethod(slog, next))
}
