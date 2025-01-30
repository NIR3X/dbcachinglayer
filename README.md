# DB Caching Layer Module - A Go-based generic, thread-safe caching and persistence layer for database records

## Overview
The `DBCL` (Database Caching Layer) module is a generic, thread-safe caching and persistence layer for database records in Go. It supports real-time caching of records and scheduled synchronization with a SQL database backend.

This implementation enables developers to manage records efficiently while abstracting typical CRUD operations and caching mechanisms.

---

## Features

- **Generic Record Handling:** Supports any custom record type by implementing the `DBCLRecord` interface.
- **Automatic Synchronization:** Periodically saves cached records to the database.
- **In-Memory Record Cache:** Reduces database queries by maintaining an in-memory cache.
- **Transactional Updates:** Ensures data consistency through SQL transactions.
- **Thread-Safe Operations:** Concurrent-safe access to records.

---

## Installation

```bash
go get github.com/NIR3X/dbcachinglayer
```

---

## Usage

### 1. Define Your Record Type
Implement the `DBCLRecord` interface for your custom record type.

```go
package main

type Note struct {
    Id      int64
    Title   string
    Content string
}

func NewNote(id int64, title, content string) *Note {
	return &Note{id, title, content}
}

func (n *Note) DBCLNewRecord(args ...interface{}) DBCLRecord {
	return NewNote(args[0].(int64), args[1].(string), args[2].(string))
}

// Implement other DBCLRecord methods...
```

### 2. Initialize the DB Caching Layer

```go
package main

import (
    "database/sql"
    "log"
    "time"

    _ "github.com/mattn/go-sqlite3"
)

func main() {
    dbcl, err := NewDBCL[*Note]("sqlite3", "notes.db", 1*time.Minute)
    if err != nil {
        log.Fatalf("Failed to create DBCL: %v", err)
    }
    defer dbcl.Close()

    note := &Note{Title: "Sample", Content: "Sample Content"}
    dbcl.InsertRecord(note)
}
```

### 3. CRUD Operations

- **Insert:**
  ```go
  dbcl.InsertRecord(note)
  ```

- **Get Record by ID:**
  ```go
  record := dbcl.GetRecord(note.Id)
  ```

- **Update:**
  ```go
  note.Title = "Updated Title"
  dbcl.UpdateRecord(note.Id, note)
  ```

- **Delete:**
  ```go
  dbcl.DeleteRecord(note.Id)
  ```

---

## Running Tests

The module includes unit tests for key operations.

```bash
go test ./...
```

---

## Example Test Output

```bash
=== RUN   TestDBCachingLayer
--- PASS: TestDBCachingLayer (0.02s)
PASS
ok      dbcachinglayer    0.021s
```

---

## License

[![GNU AGPLv3 Image](https://www.gnu.org/graphics/agplv3-155x51.png)](https://www.gnu.org/licenses/agpl-3.0.html)

This program is Free Software: You can use, study share and improve it at your
will. Specifically you can redistribute and/or modify it under the terms of the
[GNU Affero General Public License](https://www.gnu.org/licenses/agpl-3.0.html) as
published by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

