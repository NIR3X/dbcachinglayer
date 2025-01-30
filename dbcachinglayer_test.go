package dbcachinglayer

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Note struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func NewNote(id int64, title, content string) *Note {
	return &Note{id, title, content}
}

func (n *Note) DBCLNewRecord(args ...interface{}) DBCLRecord {
	return NewNote(args[0].(int64), args[1].(string), args[2].(string))
}

func (n *Note) DBCLFields() (int64, string, string) {
	return n.Id, n.Title, n.Content
}

func (n *Note) DBCLSelectAll(db *sql.DB) (*sql.Rows, error) {
	return db.Query("SELECT id, title, content FROM notes")
}

func (n *Note) DBCLInsert(tx *sql.Tx, note DBCLRecord) (sql.Result, error) {
	return tx.Exec("INSERT INTO notes (id, title, content) VALUES (?, ?)", note.(*Note).Id, note.(*Note).Title, note.(*Note).Content)
}

func (n *Note) DBCLUpdate(tx *sql.Tx, note DBCLRecord) (sql.Result, error) {
	return tx.Exec("UPDATE notes SET title = ?, content = ? WHERE id = ?", note.(*Note).Title, note.(*Note).Content, note.(*Note).Id)
}
func (n *Note) DBCLDelete(tx *sql.Tx, id int64) (sql.Result, error) {
	return tx.Exec("DELETE FROM notes WHERE id = ?", id)
}

func (n *Note) DBCLExists(tx *sql.Tx, id int64) (bool, error) {
	var exists bool
	err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM notes WHERE id = ?)", id).Scan(&exists)
	return exists, err
}

func (n *Note) DBCLScan(rows *sql.Rows) error {
	return rows.Scan(&n.Id, &n.Title, &n.Content)
}

func (n *Note) DBCLGetId() int64 {
	return n.Id
}

func (n *Note) DBCLSetId(id int64) {
	n.Id = id
}

func TestDBCachingLayer(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE notes (id INTEGER PRIMARY KEY, title TEXT, content TEXT)")
	if err != nil {
		t.Fatalf("Error creating table: %v", err)
	}

	dbcl, err := NewDBCL[*Note]("sqlite3", ":memory:", 1*time.Second)
	if err != nil {
		t.Fatalf("Error creating DBCL: %v", err)
	}
	defer dbcl.Close()

	record := NewNote(0, "Title", "Content")
	dbcl.InsertRecord(record)
	if record.Id != 1 {
		t.Fatalf("Expected record id to be 1, got %d", record.Id)
	}
	record = dbcl.GetRecord(1)
	if record == nil {
		t.Fatalf("Expected record to be found")
	}
	if record.Id != 1 {
		t.Fatalf("Expected record id to be 1, got %d", record.Id)
	}
	if record.Title != "Title" {
		t.Fatalf("Expected record title to be 'Title', got '%s'", record.Title)
	}
	if record.Content != "Content" {
		t.Fatalf("Expected record content to be 'Content', got '%s'", record.Content)
	}

	record.Title = "New Title"
	dbcl.UpdateRecord(record.Id, record)
	record = dbcl.GetRecord(record.Id)
	if record == nil {
		t.Fatalf("Expected record to be found")
	}
	if record.Id != 1 {
		t.Fatalf("Expected record id to be 1, got %d", record.Id)
	}
	if record.Title != "New Title" {
		t.Fatalf("Expected record title to be 'New Title', got '%s'", record.Title)
	}

	dbcl.DeleteRecord(1)
	record = dbcl.GetRecord(1)
	if record != nil {
		t.Fatalf("Expected record to be deleted")
	}
}
