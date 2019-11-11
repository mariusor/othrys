package boltdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mariusor/esports-calendar/calendar"
	"github.com/mariusor/esports-calendar/storage"
	bolt "go.etcd.io/bbolt"
	"time"
)

type LoggerFn func(string, ...interface{})

type repo struct {
	d       *bolt.DB
	baseURL string
	root    []byte
	path    string
	log     LoggerFn
	err     LoggerFn
}

const (
	rootBucket = "cal"
)

// Config
type Config struct {
	Path  string
	LogFn LoggerFn
	ErrFn LoggerFn
}

// New returns a new repo repository
func New(c Config) *repo {
	b := repo{
		root: []byte(rootBucket),
		path: c.Path,
		log:  func(string, ...interface{}) {},
		err:  func(string, ...interface{}) {},
	}
	if c.ErrFn != nil {
		b.err = c.ErrFn
	}
	if c.LogFn != nil {
		b.log = c.LogFn
	}

	return &b
}

func (r *repo) open() error {
	var err error
	r.d, err = bolt.Open(r.path, 0600, nil)
	if err != nil {
		return fmt.Errorf("could not open db %s %w", r.path, err)
	}
	err = r.d.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists(r.root)
		if err != nil {
			return fmt.Errorf("unable to create root bucket %s: %w", r.root, err)
		}
		if !root.Writable() {
			return fmt.Errorf("non writeable root bucket %s", r.root)
		}
		return nil
	})
	return err
}

// Close closes the boltdb database if possible.
func (r *repo) close() error {
	if r.d == nil {
		return nil
	}
	return r.d.Close()
}

// LoadEvent
func (r *repo) LoadEvent(typ string, date time.Time, id int64) calendar.Event {
	events, err := r.LoadEvents(storage.DateCursor{T: date, D: time.Hour}, typ)
	if err != nil {
		r.err("error loading events: %s", err)
	}
	for _, event := range events {
		if event.CalID == id {
			return event
		}
	}
	return calendar.Event{}
}

// LoadEvents
func (r *repo) LoadEvents(cursor storage.DateCursor, types ...string) (calendar.Events, error) {
	var err error
	err = r.open()
	if err != nil {
		return nil, err
	}
	defer r.close()
	return loadFromBucket(r.d, r.root, cursor, types...)
}

func loadFromBucketRecursive(b *bolt.Bucket, min, max []byte) calendar.Events {
	events := make(calendar.Events, 0)

	c := b.Cursor()

	first := func() ([]byte, []byte) { return c.First() }
	compare := func(k, v []byte) bool { return k != nil }
	if min != nil {
		first = func() ([]byte, []byte) { return c.Seek(min) }
	}
	if max != nil {
		compare = func(k, v []byte) bool { return k != nil && bytes.Compare(k, max) <= 0 }
	}
	for key, raw := first(); compare(key, raw); key, raw = c.Next() {
		if raw == nil {
			// this is a bucket mate: descend!
			events = append(events, loadFromBucketRecursive(b.Bucket(key), nil, nil)...)
		} else {
			ev, _ := loadItem(raw)
			if ev.IsValid() {
				events = append(events, ev)
			}
		}
	}

	return events
}

func loadFromBucket(db *bolt.DB, root []byte, cursor storage.DateCursor, types ...string) (calendar.Events, error) {
	events := make(calendar.Events, 0)

	err := db.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(root)
		if rb == nil {
			return fmt.Errorf("invalid bucket %s", root)
		}

		var err error
		for _, typ := range types {
			var b *bolt.Bucket
			min, max := getCursorPaths(cursor, []byte(typ))
			b, min, max, err = descendToLastCommonBucket(rb, min, max)
			events = append(events, loadFromBucketRecursive(b, min, max)...)
		}
		return err
	})

	return events, err
}

func loadItem(raw []byte) (calendar.Event, error) {
	ev := calendar.Event{}
	if raw == nil || len(raw) == 0 {
		// TODO(marius): log this instead of stopping the iteration and returning an error
		return ev, fmt.Errorf("empty raw item")
	}
	err := json.Unmarshal(raw, &ev)
	return ev, err
}

var pathSeparator = []byte{'/'}

func getCursorPaths(c storage.DateCursor, typ []byte) ([]byte, []byte) {
	var min, max []byte
	if c.D < 0 {
		max = itemBucketPath(typ, c.T)
		min = itemBucketPath(typ, c.T.Add(c.D))
	} else {
		min = itemBucketPath(typ, c.T)
		max = itemBucketPath(typ, c.T.Add(c.D))
	}
	return min, max
}

func itemBucketPath(typ []byte, date time.Time) []byte {
	pathEl := make([][]byte, 0)

	pathEl = append(pathEl, typ)
	pathEl = append(pathEl, []byte(date.Format("06")))
	pathEl = append(pathEl, []byte(date.Format("01")))
	pathEl = append(pathEl, []byte(date.Format("02")))
	pathEl = append(pathEl, []byte(date.Format("15")))
	pathEl = append(pathEl, []byte(date.Format("04")))

	return bytes.Join(pathEl, pathSeparator)
}

func descendToLastCommonBucket(root *bolt.Bucket, min, max []byte) (*bolt.Bucket, []byte, []byte, error) {
	minPieces := bytes.Split(min, pathSeparator)
	maxPieces := bytes.Split(max, pathSeparator)

	b := root
	lvl := 0
	// the length should be the same
	for i, k := range minPieces {
		if !bytes.Equal(k, maxPieces[i]) {
			break
		}
		cb := b.Bucket(k)
		if cb == nil {
			break
		}
		lvl = i
		b = cb
	}
	min = bytes.Join(minPieces[lvl+1:], pathSeparator)
	max = bytes.Join(maxPieces[lvl+1:], pathSeparator)
	return b, min, max, nil
}

func descendInBucket(root *bolt.Bucket, path []byte, create bool) (*bolt.Bucket, []byte, error) {
	if root == nil {
		return nil, path, fmt.Errorf("trying to descend into nil bucket")
	}
	if len(path) == 0 {
		return root, path, nil
	}
	buckets := bytes.Split(path, pathSeparator)

	lvl := 0
	b := root
	// descend the bucket tree up to the last found bucket
	for _, name := range buckets {
		lvl++
		if len(name) == 0 {
			continue
		}
		if b == nil {
			return root, path, fmt.Errorf("trying to load from nil bucket")
		}
		var cb *bolt.Bucket
		if create {
			cb, _ = b.CreateBucketIfNotExists(name)
		} else {
			cb = b.Bucket(name)
		}
		if cb == nil {
			lvl--
			break
		}
		b = cb
	}
	path = bytes.Join(buckets[lvl:], pathSeparator)

	return b, path, nil
}

// SaveEvents
func (r *repo) SaveEvents(events calendar.Events) error {
	var err error
	err = r.open()
	if err != nil {
		return err
	}
	defer r.close()

	for _, ev := range events {
		ev, err = save(r, ev)
		if err != nil {
			r.err("Error saving event %d: %s", ev.CalID, err)
		}
	}
	return err
}

// SaveEvent
func (r *repo) SaveEvent(ev calendar.Event) error {
	var err error
	err = r.open()
	if err != nil {
		return err
	}
	defer r.close()

	ev, err = save(r, ev)
	return err
}

func save(r *repo, ev calendar.Event) (calendar.Event, error) {
	path := itemBucketPath([]byte(ev.Type), ev.StartTime)

	err := r.d.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(r.root)
		if root == nil {
			return fmt.Errorf("invalid bucket %s", r.root)
		}
		if !root.Writable() {
			return fmt.Errorf("non writeable bucket %s", r.root)
		}
		b, path, err := descendInBucket(root, path, true)
		if err != nil {
			return fmt.Errorf("unable to find %s in root bucket: %w", path, err)
		}
		if !b.Writable() {
			return fmt.Errorf("non writeable bucket %s", path)
		}
		entryBytes, err := json.Marshal(ev)
		if err != nil {
			return fmt.Errorf("could not marshal object: %w", err)
		}
		objectID := []byte(fmt.Sprintf("%d", ev.CalID))
		err = b.Put(objectID[:], entryBytes)
		if err != nil {
			return fmt.Errorf("could not store encoded object: %w", err)
		}

		return nil
	})

	return ev, err
}
