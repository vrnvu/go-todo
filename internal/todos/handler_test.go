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

func testHandler(t *testing.T, tempFile *os.File) *handler {
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

	r, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, "/todos/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	r, err = http.NewRequestWithContext(context.Background(), http.MethodGet, "/todos/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w = httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, w.Code)
	}

	r, err = http.NewRequestWithContext(context.Background(), http.MethodGet, "/todos", nil)
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

	r, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/todos/1", strings.NewReader(`{"id": 1,  "description": "test","title": "test", "completed": false}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	r, err = http.NewRequestWithContext(context.Background(), http.MethodPut, "/todos/1", strings.NewReader(`{"id": 1,  "description": "test","title": "test", "completed": false}`))
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

func TestPatchWithNulls(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())
	handler := testHandler(t, tempFile)

	r, err := http.NewRequestWithContext(context.Background(), http.MethodPut, "/todos/1", strings.NewReader(`{"id": 1,  "description": "test","title": "test", "completed": false}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// we setting descrption to null, we expect it to be set to the empty value
	// we are not sending the title and completed, so they should not be modified
	r, err = http.NewRequestWithContext(context.Background(), http.MethodPatch, "/todos/1", strings.NewReader(`{"id": 1,  "description": null}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	r.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		slog.Info("w", "w", w.Body)
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	r, err = http.NewRequestWithContext(context.Background(), http.MethodGet, "/todos/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w = httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var body Todo
	err = json.Unmarshal(w.Body.Bytes(), &body)
	if err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}

	if body.Description != "" {
		t.Fatalf("expected description %s, got %s", "", body.Description)
	}

	if body.Title != "test" {
		t.Fatalf("expected title %s, got %s", "test", body.Title)
	}

	if body.Completed != false {
		t.Fatalf("expected completed %t, got %t", false, body.Completed)
	}

	if body.ID != 1 {
		t.Fatalf("expected id %d, got %d", 1, body.ID)
	}
}

func TestPatchNoFieldsToUpdate(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())
	handler := testHandler(t, tempFile)

	r, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, "/todos/1", strings.NewReader(`{"id": 1}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.Mux.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	expectedError := strings.TrimSpace(ErrNoFieldsToUpdate.Error())
	actualError := strings.TrimSpace(w.Body.String())
	if actualError != expectedError {
		t.Fatalf("expected body %q, got %q", expectedError, actualError)
	}
}
