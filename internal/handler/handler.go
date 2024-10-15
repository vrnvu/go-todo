package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vrnvu/go-todo/internal/db"
)

type XRequestIDHeader string

const XRequestIDHeaderKey XRequestIDHeader = "X-Request-ID"

type Todos struct {
	Slog *slog.Logger
	Mux  *http.ServeMux
	repo *db.Repository
}

type Config struct {
	DBFile             string
	Slog               *slog.Logger
	RequestIDGenerator func() string
}

func FromConfig(c *Config) (*Todos, error) {
	repo, err := db.NewRepository(c.DBFile)
	if err != nil {
		return nil, err
	}

	t := &Todos{Slog: c.Slog, Mux: http.NewServeMux(), repo: repo}

	t.Mux.HandleFunc("GET /todos", withBaseMiddleware(c.Slog, c.RequestIDGenerator, t.getTodos))
	t.Mux.HandleFunc("GET /todos/{id}", withBaseMiddleware(c.Slog, c.RequestIDGenerator, t.getTodo))
	t.Mux.HandleFunc("PUT /todos/{id}", withBaseMiddleware(c.Slog, c.RequestIDGenerator, t.InsertTodo))
	t.Mux.HandleFunc("DELETE /todos/{id}", withBaseMiddleware(c.Slog, c.RequestIDGenerator, t.DeleteTodo))
	return t, nil
}

func (t *Todos) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid id: `%s`", rawID), http.StatusBadRequest)
		return
	}

	if err := t.repo.DeleteTodo(id); err != nil {
		t.logError(r, fmt.Sprintf("failed to delete todo with id `%d`", id), err)
		http.Error(w, fmt.Sprintf("failed to delete todo with id `%d`", id), http.StatusInternalServerError)
		return
	}
}

func (t *Todos) InsertTodo(w http.ResponseWriter, r *http.Request) {
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

	todo := db.Todo{}
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, "failed to decode todo body", http.StatusBadRequest)
		return
	}

	if id != todo.ID {
		http.Error(w, fmt.Sprintf("id in path `%d` and body `%d` do not match", id, todo.ID), http.StatusBadRequest)
		return
	}

	if err := t.repo.InsertTodo(todo); err != nil {
		t.logError(r, "failed to insert todo", err)
		http.Error(w, "failed to insert todo", http.StatusInternalServerError)
		return
	}
}

func (t *Todos) getTodos(w http.ResponseWriter, r *http.Request) {
	todos, err := t.repo.GetTodos()
	if err != nil {
		t.logError(r, "failed to get todos", err)
		http.Error(w, "failed to get todos", http.StatusInternalServerError)
		return
	}

	t.writeJSON(w, r, todos)
}

func (t *Todos) getTodo(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		t.logError(r, fmt.Sprintf("failed to convert id `%s` to uint", rawID), err)
		http.Error(w, fmt.Sprintf("invalid id: `%s`", rawID), http.StatusBadRequest)
		return
	}

	todo, err := t.repo.GetTodo(id)
	if err != nil {
		var notFoundErr db.ErrNotFound
		if errors.As(err, &notFoundErr) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		t.logError(r, fmt.Sprintf("failed to get todo with id `%d`", id), err)
		http.Error(w, fmt.Sprintf("failed to get todo with id `%d`", id), http.StatusInternalServerError)
		return
	}

	t.writeJSON(w, r, todo)
}

func (t *Todos) writeJSON(w http.ResponseWriter, r *http.Request, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", fromContext(r, XRequestIDHeaderKey))

	if err := json.NewEncoder(w).Encode(data); err != nil {
		t.logError(r, "failed to encode data", err)
		http.Error(w, "failed to encode data", http.StatusInternalServerError)
		return
	}
}

func (t *Todos) logError(r *http.Request, message string, err error) {
	t.Slog.Error(message, "error", err, string(XRequestIDHeaderKey), fromContext(r, XRequestIDHeaderKey))
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
