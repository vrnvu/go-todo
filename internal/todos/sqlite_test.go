package todos

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"
)

func exampleTodo() Todo {
	return Todo{
		ID:          1,
		Title:       "Todo 1",
		Description: "Description 1",
		Completed:   false,
	}
}

func TestGetTodoNotFound(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())

	db, err := NewDB(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	todo, err := db.Get(context.Background(), 1)
	if todo != nil {
		t.Fatalf("expected todo to be nil, got %v", todo)
	}

	var notFound ErrNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected error to be ErrNotFound, got %v", err)
	}
}

func TestGetTodo(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())

	db, err := NewDB(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	want := exampleTodo()
	err = db.Insert(context.Background(), want)
	if err != nil {
		t.Fatalf("failed to insert todo: %v", err)
	}

	got, err := db.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("expected todo to be %v, got %v", want, got)
	}
}

func TestDeleteNonExistentTodo(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())

	db, err := NewDB(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	err = db.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("failed to delete todo: %v", err)
	}
}

func TestDeleteTodo(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())

	db, err := NewDB(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	want := exampleTodo()
	err = db.Insert(context.Background(), want)
	if err != nil {
		t.Fatalf("failed to insert todo: %v", err)
	}

	err = db.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("failed to delete todo: %v", err)
	}

	got, err := db.Get(context.Background(), 1)
	if got != nil {
		t.Fatalf("expected todo to be nil, got %v", got)
	}

	var notFound ErrNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected error to be ErrNotFound, got %v", err)
	}
}

func TestGetTodos(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())

	db, err := NewDB(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	todos, err := db.GetAll(context.Background())
	if err != nil {
		t.Fatalf("failed to get todos: %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("expected 0 todos, got %d", len(todos))
	}

	err = db.Insert(context.Background(), exampleTodo())
	if err != nil {
		t.Fatalf("failed to insert todo: %v", err)
	}

	todos, err = db.GetAll(context.Background())
	if err != nil {
		t.Fatalf("failed to get todos: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
}

func TestPatchTodo(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())

	db, err := NewDB(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	todo := Todo{
		ID:          1,
		Title:       "Todo 1",
		Description: "Description 1",
		Completed:   true,
	}

	err = db.Insert(context.Background(), todo)
	if err != nil {
		t.Fatalf("failed to insert todo: %v", err)
	}

	patch := NewTodoPatch()
	body := strings.NewReader(`{"id": 1,  "description": null, "completed": null}`)
	if err := json.NewDecoder(body).Decode(&patch); err != nil {
		t.Fatalf("failed to decode todo body: %v", err)
	}
	slog.Info("patch", "patch", patch)

	err = db.Patch(context.Background(), patch)
	if err != nil {
		t.Fatalf("failed to patch todo: %v", err)
	}

	got, err := db.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	slog.Info("got", "got", got)

	if got.Description != "" {
		t.Fatalf("expected description to be empty, got %s", got.Description)
	}

	if got.Completed != false {
		t.Fatalf("expected completed to be false, got %t", got.Completed)
	}

	if got.Title != "Todo 1" {
		t.Fatalf("expected title to be Todo 1, got %s", got.Title)
	}
}
