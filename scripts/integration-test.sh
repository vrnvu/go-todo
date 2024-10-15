#!/bin/bash

# Function to check HTTP status code
check_status() {
  if [ "$1" -ne "$2" ]; then
    echo "Expected status $2 but got $1"
    exit 1
  fi
}

# Function to check JSON response
check_json() {
  if [ "$1" != "$2" ]; then
    echo "Expected JSON $2 but got $1"
    exit 1
  fi
}

# Create or Update Todo with ID 1
response=$(curl -s -o /dev/null -w "%{http_code}" -X PUT http://localhost:8080/todos/1 \
     -H "Content-Type: application/json" \
     -d '{"id": 1, "title": "First Todo", "description": "This is the first todo", "completed": false}')
check_status $response 200

# Create or Update Todo with ID 2
response=$(curl -s -o /dev/null -w "%{http_code}" -X PUT http://localhost:8080/todos/2 \
     -H "Content-Type: application/json" \
     -d '{"id": 2, "title": "Second Todo", "description": "This is the second todo", "completed": false}')
check_status $response 200

# Get All Todos
response=$(curl -s -w "%{http_code}" -o todos.json -X GET http://localhost:8080/todos)
check_status ${response: -3} 200
todos=$(jq length todos.json)
check_json $todos 2

# Delete Todo with ID 1
response=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE http://localhost:8080/todos/1)
check_status $response 200

# Get All Todos after deletion
response=$(curl -s -w "%{http_code}" -o todos.json -X GET http://localhost:8080/todos)
check_status ${response: -3} 200
todos=$(jq length todos.json)
check_json $todos 1

# Get Todo with ID 2
response=$(curl -s -w "%{http_code}" -o todo.json -X GET http://localhost:8080/todos/2)
check_status ${response: -3} 200
title=$(jq -r '.title' todo.json)
check_json "$title" "Second Todo"

# Update Todo with ID 2 to completed true
response=$(curl -s -o /dev/null -w "%{http_code}" -X PUT http://localhost:8080/todos/2 \
     -H "Content-Type: application/json" \
     -d '{"id": 2, "title": "Second Todo", "description": "This is the second todo updated", "completed": true}')
check_status $response 200

# Get Todo with ID 2 again
response=$(curl -s -w "%{http_code}" -o todo.json -X GET http://localhost:8080/todos/2)
check_status ${response: -3} 200
completed=$(jq -r '.completed' todo.json)
check_json "$completed" "true"

echo "All tests passed!"