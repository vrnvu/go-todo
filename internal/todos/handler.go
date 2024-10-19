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

const (
	headerContentType    = "Content-Type"
	valueContentTypeJSON = "application/json"
	headerXRequestID     = "X-Request-ID"
)

type xRequestIDHeader string

const xRequestIDHeaderKey xRequestIDHeader = headerXRequestID

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
	h.Mux.HandleFunc("PATCH /todos/{id}", withBaseMiddleware(c.Slog, c.RequestIDGenerator, h.patch))
	h.Mux.HandleFunc("DELETE /todos/{id}", withBaseMiddleware(c.Slog, c.RequestIDGenerator, h.delete))
	return h, nil
}

func health(_ http.ResponseWriter, _ *http.Request) {}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := fromPathTodoID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.db.Delete(r.Context(), id); err != nil {
		h.logError(r, http.StatusText(http.StatusInternalServerError), err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

// patch handles patching a todo with a potentially partial JSON body.
//
// Example:
// { id: 1, "title": "new title" } will only update the title.
// { id: 1, "completed": true } will only update the completed field.
//
// id is required and must match the id in the path.
//
// If the Todo has a field set to null, it will be set to the empty value.
//
// Example:
// { id: 1, "completed": null } will set completed to false.
func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	if err := assertHeaderValueIs(r, headerContentType, valueContentTypeJSON); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := fromPathTodoID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	patch := NewTodoPatch()
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if id != patch.ID {
		http.Error(w, fmt.Sprintf("id in path `%d` and body `%d` do not match", id, patch.ID), http.StatusBadRequest)
		return
	}

	if err := h.db.Patch(r.Context(), patch); err != nil {
		if errors.Is(err, ErrNoFieldsToUpdate) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		h.logError(r, http.StatusText(http.StatusInternalServerError), err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) insert(w http.ResponseWriter, r *http.Request) {
	if err := assertHeaderValueIs(r, headerContentType, valueContentTypeJSON); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := fromPathTodoID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		h.logError(r, http.StatusText(http.StatusInternalServerError), err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) getAll(w http.ResponseWriter, r *http.Request) {
	todos, err := h.db.GetAll(r.Context())
	if err != nil {
		h.logError(r, http.StatusText(http.StatusInternalServerError), err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, r, todos)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := fromPathTodoID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	todo, err := h.db.Get(r.Context(), id)
	if err != nil {
		var notFoundErr ErrNotFound
		if errors.As(err, &notFoundErr) {
			http.Error(w, notFoundErr.Error(), http.StatusNotFound)
			return
		}

		h.logError(r, http.StatusText(http.StatusInternalServerError), err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, r, todo)
}

func (h *Handler) writeJSON(w http.ResponseWriter, r *http.Request, data any) {
	w.Header().Set(headerContentType, valueContentTypeJSON)
	w.Header().Set(headerXRequestID, fromContext(r, xRequestIDHeaderKey))

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logError(r, http.StatusText(http.StatusInternalServerError), err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) logError(r *http.Request, message string, err error) {
	h.Slog.Error(message, "error", err, "method", r.Method, "path", r.URL.Path, headerXRequestID, fromContext(r, xRequestIDHeaderKey))
}

func fromContext(r *http.Request, key any) string {
	return r.Context().Value(key).(string)
}

func withRequestID(requestIDGenerator func() string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := requestIDGenerator()
		ctx := context.WithValue(r.Context(), xRequestIDHeaderKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func withLoggingMethod(slog *slog.Logger, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("", "method", r.Method, "path", r.URL.Path, "requestID", fromContext(r, xRequestIDHeaderKey))
		next.ServeHTTP(w, r)
	}
}

func withBaseMiddleware(slog *slog.Logger, requestIDGenerator func() string, next http.HandlerFunc) http.HandlerFunc {
	return withRequestID(requestIDGenerator, withLoggingMethod(slog, next))
}

func assertHeaderValueIs(r *http.Request, header string, value string) error {
	if r.Header.Get(header) != value {
		return fmt.Errorf("invalid header `%s` value: got `%s`, use `%s`", header, r.Header.Get(header), value)
	}
	return nil
}

func fromPathTodoID(r *http.Request) (int, error) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		return 0, fmt.Errorf("invalid id: `%s`", rawID)
	}
	return id, nil
}
