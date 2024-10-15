package db

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

func exampleTodo() Todo {
	return Todo{
		ID:          1,
		Title:       "Todo 1",
		Description: "Description 1",
		Completed:   false,
	}
}

func testTempFile(t *testing.T) *os.File {
	name := fmt.Sprintf("test-%d.db", time.Now().UnixNano())
	tempFile, err := os.CreateTemp("", name)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return tempFile
}

func TestGetTodoNotFound(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())

	repository, err := NewRepository(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	todo, err := repository.GetTodo(1)
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

	repository, err := NewRepository(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	want := exampleTodo()
	err = repository.InsertTodo(want)
	if err != nil {
		t.Fatalf("failed to insert todo: %v", err)
	}

	got, err := repository.GetTodo(1)
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

	repository, err := NewRepository(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	err = repository.DeleteTodo(1)
	if err != nil {
		t.Fatalf("failed to delete todo: %v", err)
	}
}

func TestDeleteTodo(t *testing.T) {
	t.Parallel()
	tempFile := testTempFile(t)
	defer os.Remove(tempFile.Name())

	repository, err := NewRepository(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	want := exampleTodo()
	err = repository.InsertTodo(want)
	if err != nil {
		t.Fatalf("failed to insert todo: %v", err)
	}

	err = repository.DeleteTodo(1)
	if err != nil {
		t.Fatalf("failed to delete todo: %v", err)
	}

	got, err := repository.GetTodo(1)
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

	repository, err := NewRepository(tempFile.Name())
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}

	todos, err := repository.GetTodos()
	if err != nil {
		t.Fatalf("failed to get todos: %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("expected 0 todos, got %d", len(todos))
	}

	err = repository.InsertTodo(exampleTodo())
	if err != nil {
		t.Fatalf("failed to insert todo: %v", err)
	}

	todos, err = repository.GetTodos()
	if err != nil {
		t.Fatalf("failed to get todos: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
}
