package db

import (
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

type Repository struct {
	db             *sql.DB
	stmtInsertTodo *sql.Stmt
	stmtGetTodo    *sql.Stmt
	stmtGetTodos   *sql.Stmt
	stmtDeleteTodo *sql.Stmt
}

func NewRepository(dbFile string) (*Repository, error) {
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

	return &Repository{db: db, stmtInsertTodo: insertStmt, stmtGetTodo: getStmt, stmtGetTodos: getAllStmt, stmtDeleteTodo: deleteStmt}, nil
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("todo `%d` not found", e.ID)
}

func (r *Repository) InsertTodo(todo Todo) error {
	_, err := r.stmtInsertTodo.Exec(todo.ID, todo.Title, todo.Description, todo.Completed)
	return err
}

func (r *Repository) DeleteTodo(id int) error {
	_, err := r.stmtDeleteTodo.Exec(id)
	return err
}

func (r *Repository) GetTodo(id int) (*Todo, error) {
	row := r.stmtGetTodo.QueryRow(id)
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

func (r *Repository) GetTodos() ([]Todo, error) {
	rows, err := r.stmtGetTodos.Query()
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
