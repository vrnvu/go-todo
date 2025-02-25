package todos

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	// Register the SQLite driver with the database/sql package
	_ "github.com/mattn/go-sqlite3"
)

type TodoPatch struct {
	data map[string]any

	ID          int
	Title       *string
	Description *string
	Completed   *bool
}

func NewTodoPatch() TodoPatch {
	return TodoPatch{data: make(map[string]any)}
}

func (tp *TodoPatch) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &tp.data)
	if err != nil {
		return err
	}

	if ID, ok := tp.data["id"]; ok {
		if ID == nil {
			return fmt.Errorf("id is required")
		}
		floatID, ok := ID.(float64)
		if !ok {
			return fmt.Errorf("id is not a float64")
		}

		tp.ID = int(floatID)
	}

	if title, ok := tp.data["title"]; ok {
		if title == nil {
			defaultTitle := ""
			tp.Title = &defaultTitle
		} else {
			tp.Title, ok = title.(*string)
			if !ok {
				return fmt.Errorf("title is not a string")
			}
		}
	}

	if description, ok := tp.data["description"]; ok {
		if description == nil {
			defaultDescription := ""
			tp.Description = &defaultDescription
		} else {
			tp.Description, ok = description.(*string)
			if !ok {
				return fmt.Errorf("description is not a string")
			}
		}
	}

	if completed, ok := tp.data["completed"]; ok {
		if completed == nil {
			defaultCompleted := false
			tp.Completed = &defaultCompleted
		} else {
			tp.Completed, ok = completed.(*bool)
			if !ok {
				return fmt.Errorf("completed is not a boolean")
			}
		}
	}

	return nil
}

type Todo struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

type ErrNotFound struct {
	ID int
}

var ErrNoFieldsToUpdate = errors.New("no fields to update")

type DB struct {
	db         *sql.DB
	stmtInsert *sql.Stmt
	stmtGet    *sql.Stmt
	stmtGetAll *sql.Stmt
	stmtDelete *sql.Stmt
}

func NewDB(dbFile string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS todos (id INTEGER PRIMARY KEY, title TEXT, description TEXT, completed BOOLEAN)")
	if err != nil {
		return nil, err
	}

	insertStmt, err := db.Prepare("INSERT OR REPLACE INTO todos (id, title, description, completed) VALUES (?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}

	getStmt, err := db.Prepare("SELECT id, title, description, completed FROM todos WHERE id = ?")
	if err != nil {
		return nil, err
	}

	getAllStmt, err := db.Prepare("SELECT id, title, description, completed FROM todos")
	if err != nil {
		return nil, err
	}

	deleteStmt, err := db.Prepare("DELETE FROM todos WHERE id = ?")
	if err != nil {
		return nil, err
	}

	return &DB{db: db, stmtInsert: insertStmt, stmtGet: getStmt, stmtGetAll: getAllStmt, stmtDelete: deleteStmt}, nil
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("todo `%d` not found", e.ID)
}

func (t *DB) Insert(ctx context.Context, todo Todo) error {
	_, err := t.stmtInsert.ExecContext(ctx, todo.ID, todo.Title, todo.Description, todo.Completed)
	return err
}

func (t *DB) Delete(ctx context.Context, id int) error {
	_, err := t.stmtDelete.ExecContext(ctx, id)
	return err
}

func (t *DB) Get(ctx context.Context, id int) (*Todo, error) {
	row := t.stmtGet.QueryRowContext(ctx, id)
	var todo Todo
	err := row.Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Completed)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound{ID: id}
		}
		return nil, err
	}
	return &todo, nil
}

func (t *DB) GetAll(ctx context.Context) ([]Todo, error) {
	rows, err := t.stmtGetAll.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Completed)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}

	return todos, nil
}

func (t *DB) Patch(ctx context.Context, patch TodoPatch) error {
	var queryBuilder strings.Builder
	args := []any{}

	queryBuilder.WriteString("UPDATE todos SET ")

	if patch.Title != nil {
		queryBuilder.WriteString("title = ?, ")
		args = append(args, patch.Title)
	}

	if patch.Description != nil {
		queryBuilder.WriteString("description = ?, ")
		args = append(args, patch.Description)
	}

	if patch.Completed != nil {
		queryBuilder.WriteString("completed = ?, ")
		args = append(args, patch.Completed)
	}

	if len(args) == 0 {
		return ErrNoFieldsToUpdate
	}

	query := queryBuilder.String()
	query = query[:len(query)-2] // Remove trailing comma and space
	query += " WHERE id = ?"
	args = append(args, patch.data["id"])

	stmt, err := t.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, args...)
	return err
}
