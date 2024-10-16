package todos

import (
	"context"
	"database/sql"
	"fmt"

	// Register the SQLite driver with the database/sql package
	_ "github.com/mattn/go-sqlite3"
)

type Todo struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

type ErrNotFound struct {
	ID int
}

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
