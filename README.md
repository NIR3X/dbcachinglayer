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
- **By-Columns Record Indexing:** Efficient retrieval of records using custom column values as lookup keys.

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
	Id int64
	Title string
	Content string
}

func NewNote(id int64, title, content string) *Note {
	return &Note{id, title, content}
}

func (n *Note) DBCLClone() DBCLRecord {
	return NewNote(n.Id, n.Title, n.Content)
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
	dbcl, err := NewDBCL[*Note]("sqlite3", "notes.db", 15*time.Second, nil)
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

## By-Columns Feature

The `byColumns` feature is a powerful way to maintain an additional in-memory lookup table for efficient record retrieval based on specific column values (such as `title`). This helps avoid full scans and speeds up searches when using frequently accessed columns as indexes.

You can define a custom modify callback function to track changes in specific columns:

```go
dbcl, err := NewDBCL[*Note]("sqlite3", ":memory:", 15*time.Second, func(byColumns map[string]interface{}, id int64, oldRecord, newRecord *Note) {
	byTitle, ok := byColumns["title"].(map[string]*Note)
	if !ok {
		byTitle = make(map[string]*Note)
		byColumns["title"] = byTitle
	}
	if oldRecord != nil {
		delete(byTitle, oldRecord.Title)
	}
	if newRecord != nil {
		byTitle[newRecord.Title] = newRecord
	}
})
```

### Retrieve Record by Column

To efficiently get a record by a custom column value (e.g., title), use `GetRecordByColumn`:

```go
record := GetRecordByColumn(dbcl, "title", "Sample Title")
if record != nil {
	log.Printf("Found record: %+v\n", record)
} else {
	log.Println("Record not found")
}
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
=== RUN TestDBCachingLayer
--- PASS: TestDBCachingLayer (0.02s)
PASS
ok dbcachinglayer 0.021s
```

---

## License

[![GNU AGPLv3 Image](https://www.gnu.org/graphics/agplv3-155x51.png)](https://www.gnu.org/licenses/agpl-3.0.html)

This program is Free Software: You can use, study share and improve it at your
will. Specifically you can redistribute and/or modify it under the terms of the
[GNU Affero General Public License](https://www.gnu.org/licenses/agpl-3.0.html) as
published by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
