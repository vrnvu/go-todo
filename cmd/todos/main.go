package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/jaevor/go-nanoid"
	"github.com/vrnvu/go-todo/internal/handler"
)

func fromEnvFile() string {
	file, ok := os.LookupEnv("DB_FILE")
	if !ok {
		file = "todos.db"
	}
	return file
}

func fromEnvPort() string {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}
	return port
}

func fromEnvSlog() (*slog.Logger, error) {
	logLevel := slog.LevelInfo
	if v, ok := os.LookupEnv("LOG_LEVEL"); ok {
		switch v {
		case "debug":
			logLevel = slog.LevelDebug
		case "info":
			logLevel = slog.LevelInfo
		case "warn":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		default:
			return nil, fmt.Errorf("invalid log level: `%s`, try: [debug, info, warn, error]", v)
		}
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})), nil
}

func main() {
	dbFile := fromEnvFile()
	port := fromEnvPort()
	slog, err := fromEnvSlog()
	if err != nil {
		panic(err)
	}

	requestIDGenerator, err := nanoid.Canonic()
	if err != nil {
		panic(err)
	}

	todosHandler, err := handler.FromConfig(&handler.Config{
		DBFile:             dbFile,
		Slog:               slog,
		RequestIDGenerator: requestIDGenerator,
	})
	if err != nil {
		panic(err)
	}

	slog.Info("starting server", "port", port)
	if err := http.ListenAndServe(":"+port, todosHandler.Mux); err != nil {
		panic(err)
	}
}
