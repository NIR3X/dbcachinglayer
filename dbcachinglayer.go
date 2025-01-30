package dbcachinglayer

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

type DBCLRecord interface {
	DBCLNewRecord(...interface{}) DBCLRecord
	DBCLSelectAll(*sql.DB) (*sql.Rows, error)
	DBCLInsert(*sql.Tx, DBCLRecord) (sql.Result, error)
	DBCLUpdate(*sql.Tx, DBCLRecord) (sql.Result, error)
	DBCLDelete(*sql.Tx, int64) (sql.Result, error)
	DBCLExists(*sql.Tx, int64) (bool, error)
	DBCLScan(*sql.Rows) error
	DBCLGetId() int64
	DBCLSetId(int64)
}

type DBCL[Record DBCLRecord] struct {
	db           *sql.DB
	ticker       *time.Ticker
	wg           sync.WaitGroup
	mtx          sync.Mutex
	stop         chan bool
	keyRecords   []int64
	nextRecordId int64
	records      map[int64]Record
	writeCache   map[int64][]Record
}

func NewDBCL[Record DBCLRecord](driverName, dataSourceName string, interval time.Duration) (*DBCL[Record], error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	s := &DBCL[Record]{
		db:           db,
		ticker:       time.NewTicker(interval),
		stop:         make(chan bool),
		keyRecords:   make([]int64, 0),
		nextRecordId: 1,
		records:      make(map[int64]Record),
		writeCache:   make(map[int64][]Record),
	}
	s.loadRecords()
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ticker.C:
				if err := s.saveRecords(); err != nil {
					log.Printf("Error saving records: %v", err)
				}
			case <-s.stop:
				return
			}
		}
	}()

	return s, nil
}

func (d *DBCL[Record]) loadRecords() error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	var r Record
	rows, err := r.DBCLSelectAll(d.db)
	if err != nil {
		return fmt.Errorf("error querying records: %w", err)
	}
	defer rows.Close()

	d.keyRecords = make([]int64, 0)
	d.records = make(map[int64]Record)

	for rows.Next() {
		var record Record
		if err := record.DBCLScan(rows); err != nil {
			return fmt.Errorf("error scanning record: %w", err)
		}

		d.records[record.DBCLGetId()] = record
		d.keyRecords = append(d.keyRecords, record.DBCLGetId())
		d.nextRecordId = record.DBCLGetId() + 1
	}

	return nil
}

func (d *DBCL[Record]) saveRecords() error {
	d.mtx.Lock()
	writeCache := d.writeCache
	d.writeCache = make(map[int64][]Record)
	d.mtx.Unlock()

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}

	defer func() {
		rollbackChanges := func() {
			tx.Rollback()
			d.mtx.Lock()
			for id, recordChanges := range writeCache {
				d.writeCache[id] = append(recordChanges, d.writeCache[id]...)
			}
			d.mtx.Unlock()
		}

		if p := recover(); p != nil {
			rollbackChanges()
			panic(p)
		} else if err != nil {
			rollbackChanges()
		} else {
			err = tx.Commit()
		}
	}()

	for id, recordChanges := range writeCache {
		for _, recordChange := range recordChanges {
			var r, zero Record
			if any(recordChange) == any(zero) {
				_, err = r.DBCLDelete(tx, id)
				if err != nil {
					return fmt.Errorf("error deleting record with ID %v: %w", id, err)
				}
				log.Printf("Deleted record with ID %v", id)
			} else {
				var exists bool
				exists, err = r.DBCLExists(tx, id)
				if err != nil {
					return fmt.Errorf("error checking existence of record with ID %v: %w", id, err)
				}
				if exists {
					_, err = r.DBCLUpdate(tx, recordChange)
					if err != nil {
						return fmt.Errorf("error updating record with ID %v: %w", id, err)
					}
					log.Printf("Updated record with ID %v", id)
				} else {
					_, err = r.DBCLInsert(tx, recordChange)
					if err != nil {
						return fmt.Errorf("error inserting record with ID %v: %w", id, err)
					}
					log.Printf("Inserted new record with ID %v", id)
				}
			}
		}
	}

	return err
}

func (d *DBCL[Record]) GetRecord(id int64) Record {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	return d.records[id]
}

func (d *DBCL[Record]) GetRecordsRange(offset, limit int64) []Record {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	records := make([]Record, 0)
	keyRecordsLength := int64(len(d.keyRecords))

	if offset < keyRecordsLength {
		cuttingLength := min(keyRecordsLength, offset+limit)

		for _, id := range d.keyRecords[offset:cuttingLength] {
			records = append(records, d.records[id])
		}
	}

	return records
}

func binarySearch(arr []int64, target int64) int {
	left, right := 0, len(arr)-1

	for left <= right {
		mid := left + (right-left)/2
		if arr[mid] == target {
			return mid
		}
		if arr[mid] < target {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return -1
}

func (d *DBCL[Record]) modifyRecord(id int64, record Record) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if id == 0 {
		id = d.nextRecordId
	}
	_, recordExists := d.records[id]
	var zero Record
	if any(record) == any(zero) {
		if recordExists {
			i := binarySearch(d.keyRecords, id)
			if i != -1 {
				newLen := len(d.keyRecords) - 1
				d.keyRecords[i] = d.keyRecords[newLen]
				d.keyRecords = d.keyRecords[:newLen]
				sort.Slice(d.keyRecords, func(i, j int) bool {
					return d.keyRecords[i] < d.keyRecords[j]
				})
			}
			delete(d.records, id)
			d.writeCache[id] = append(d.writeCache[id], zero)
		}
	} else {
		record.DBCLSetId(id)
		if !recordExists {
			d.nextRecordId++
			d.keyRecords = append(d.keyRecords, id)
			sort.Slice(d.keyRecords, func(i, j int) bool {
				return d.keyRecords[i] < d.keyRecords[j]
			})
		}
		d.records[id] = record
		d.writeCache[id] = append(d.writeCache[id], record)
	}
}

func (d *DBCL[Record]) InsertRecord(record Record) {
	d.modifyRecord(0, record)
}

func (d *DBCL[Record]) UpdateRecord(id int64, record Record) {
	record.DBCLSetId(id)
	d.modifyRecord(id, record)
}

func (d *DBCL[Record]) DeleteRecord(id int64) {
	var zero Record
	d.modifyRecord(id, zero)
}

func (d *DBCL[Record]) Close() error {
	d.ticker.Stop()
	close(d.stop)
	d.wg.Wait()
	return d.db.Close()
}
