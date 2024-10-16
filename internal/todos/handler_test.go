package todos

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func testHandler(t *testing.T, tempFile *os.File) *Handler {
	handler, err := FromConfig(&Config{
		DBFile: tempFile.Name(),
		Slog:   slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		RequestIDGenerator: func() string {
			return "123"
		},
	})
	if err != nil {
		t.Fatalf("failed to create handler: %v", err)
	}
	return handler
}

func testTempFile(t *testing.T) *os.File {
	name := fmt.Sprintf("test-%d.db", time.Now().UnixNano())
	tempFile, err := os.CreateTemp("", name)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return tempFile
}

func TestTodosHandlerFailure(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())
	handler := testHandler(t, tempFile)

	r, err := http.NewRequestWithContext(context.Background(), "DELETE", "/todos/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	r, err = http.NewRequestWithContext(context.Background(), "GET", "/todos/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w = httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, w.Code)
	}

	r, err = http.NewRequestWithContext(context.Background(), "GET", "/todos", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w = httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var body []Todo
	err = json.Unmarshal(w.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}

	if len(body) != 0 {
		t.Fatalf("expected body to be %v, got %v", []Todo{}, body)
	}
}

func TestContentType(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())
	handler := testHandler(t, tempFile)

	r, err := http.NewRequestWithContext(context.Background(), "PUT", "/todos/1", strings.NewReader(`{"id": 1,  "description": "test","title": "test", "completed": false}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	r, err = http.NewRequestWithContext(context.Background(), "PUT", "/todos/1", strings.NewReader(`{"id": 1,  "description": "test","title": "test", "completed": false}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	r.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}
}
