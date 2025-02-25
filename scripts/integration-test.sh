#!/bin/bash
set -e  # Exit immediately if a command exits with a non-zero status

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is not installed. Please install it to run this script."
    exit 1
fi

# Set the base URL based on the environment
case "$ENV" in
  dev)
    BASE_URL="http://localhost:8080"
    ;;
  pre)
    BASE_URL="http://pre.example.com"
    ;;
  pro)
    BASE_URL="http://pro.example.com"
    ;;
  *)
    echo "Using default environment (dev)"
    BASE_URL="http://localhost:8080"
    ;;
esac

echo "Running integration tests against $BASE_URL"

# Create a temporary directory for test artifacts
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Function to check HTTP status code
check_status() {
  if [ "$1" -ne "$2" ]; then
    echo "ERROR: Expected status $2 but got $1"
    exit 1
  else
    echo "✓ Status check passed: $1"
  fi
}

# Function to check JSON response
check_json() {
  if [ "$1" != "$2" ]; then
    echo "ERROR: Expected JSON value '$2' but got '$1'"
    exit 1
  else
    echo "✓ JSON check passed: $1"
  fi
}

echo "Test: Create Todo with ID 1"
response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/response.txt" -X PUT "$BASE_URL/todos/1" \
     -H "Content-Type: application/json" \
     -d '{"id": 1, "title": "First Todo", "description": "This is the first todo", "completed": false}')
check_status $response 200

echo "Test: Create Todo with ID 2"
response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/response.txt" -X PUT "$BASE_URL/todos/2" \
     -H "Content-Type: application/json" \
     -d '{"id": 2, "title": "Second Todo", "description": "This is the second todo", "completed": false}')
check_status $response 200

echo "Test: Get All Todos"
response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/todos.json" -X GET "$BASE_URL/todos")
check_status ${response: -3} 200
todos=$(jq length "$TEMP_DIR/todos.json")
check_json "$todos" "2"

echo "Test: Delete Todo with ID 1"
response=$(curl -s -o "$TEMP_DIR/response.txt" -w "%{http_code}" -X DELETE "$BASE_URL/todos/1")
check_status $response 200

echo "Test: Get All Todos after deletion"
response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/todos.json" -X GET "$BASE_URL/todos")
check_status ${response: -3} 200
todos=$(jq length "$TEMP_DIR/todos.json")
check_json "$todos" "1"

echo "Test: Get Todo with ID 2"
response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/todo.json" -X GET "$BASE_URL/todos/2")
check_status ${response: -3} 200
title=$(jq -r '.title' "$TEMP_DIR/todo.json")
check_json "$title" "Second Todo"

echo "Test: Update Todo with ID 2 to completed true"
response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/response.txt" -X PUT "$BASE_URL/todos/2" \
     -H "Content-Type: application/json" \
     -d '{"id": 2, "title": "Second Todo", "description": "This is the second todo updated", "completed": true}')
check_status $response 200

echo "Test: Get Todo with ID 2 again to verify update"
response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/todo.json" -X GET "$BASE_URL/todos/2")
check_status ${response: -3} 200
completed=$(jq '.completed' "$TEMP_DIR/todo.json")
check_json "$completed" "true"
description=$(jq -r '.description' "$TEMP_DIR/todo.json")
check_json "$description" "This is the second todo updated"

echo "Test: Patch Todo with ID 2 to set description to null"
response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/response.txt" -X PATCH "$BASE_URL/todos/2" \
     -H "Content-Type: application/json" \
     -d '{"id": 2,"description": null}')
check_status $response 200

echo "Test: Get Todo with ID 2 to verify description is empty"
response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/todo.json" -X GET "$BASE_URL/todos/2")
check_status ${response: -3} 200
description=$(jq -r '.description' "$TEMP_DIR/todo.json")
check_json "$description" ""

echo "Test: Try to get non-existent Todo"
response=$(curl -s -w "%{http_code}" -o "$TEMP_DIR/response.txt" -X GET "$BASE_URL/todos/999")
check_status $response 404

echo "All tests passed successfully!"
exit 0
