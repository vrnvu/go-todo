# go-todos

A simple todos API written in Go using a SQLite database.

```sh
# Create or Update Todo with ID 1
curl -X PUT http://localhost:8080/todos/1 \
     -H "Content-Type: application/json" \
     -d '{"id": 1, "title": "First Todo", "description": "This is the first todo", "completed": false}'

# Create or Update Todo with ID 2
curl -X PUT http://localhost:8080/todos/2 \
     -H "Content-Type: application/json" \
     -d '{"id": 2, "title": "Second Todo", "description": "This is the second todo", "completed": false}'

# Get All Todos
curl -X GET http://localhost:8080/todos

# Delete Todo with ID 1
curl -X DELETE http://localhost:8080/todos/1

# Get All Todos after deletion
curl -X GET http://localhost:8080/todos

# Get Todo with ID 2
curl -X GET http://localhost:8080/todos/2

# Update Todo with ID 2 to completed true
curl -X PUT http://localhost:8080/todos/2 \
     -H "Content-Type: application/json" \
     -d '{"id": 2, "title": "Second Todo", "description": "This is the second todo updated", "completed": true}'

# Get Todo with ID 2 again
curl -X GET http://localhost:8080/todos/2

# Patch Todo with ID 2 to set description to null
curl -X PATCH http://localhost:8080/todos/2 \
     -H "Content-Type: application/json" \
     -d '{"id": 2,""description": null}'

# Get Todo with ID 2 to verify description is an empty string
curl -X GET http://localhost:8080/todos/2
```

# Todo

- server
- opentelemtry
- render buffer pool