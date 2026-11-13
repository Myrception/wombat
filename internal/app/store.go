package app

import (
	"errors"
	"path/filepath"

	badger "github.com/dgraph-io/badger/v4"
)

var (
	errKeyNotFound = badger.ErrKeyNotFound
	// errNoStore is returned by all store operations when the database could
	// not be opened (e.g. another Wombat instance holds the lock).
	errNoStore = errors.New("app: database is not available")
)

type dblogger = badger.Logger

type store struct {
	db *badger.DB
}

func newStore(path string, l dblogger) (*store, error) {
	dbPath := filepath.Join(path, "db")
	opts := badger.DefaultOptions(dbPath)
	opts = opts.WithLogger(l)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &store{db}, nil
}

func (s *store) get(key []byte) (val []byte, rtnErr error) {
	if s == nil || s.db == nil {
		return nil, errNoStore
	}
	rtnErr = s.db.View(func(txn *badger.Txn) error {
		data, err := txn.Get(key)
		if err != nil {
			return err
		}
		val, err = data.ValueCopy(val)
		return err
	})
	return val, rtnErr
}

func (s *store) set(key, val []byte) error {
	if s == nil || s.db == nil {
		return errNoStore
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, val)
	})
}

func (s *store) del(key []byte) error {
	if s == nil || s.db == nil {
		return errNoStore
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

func (s *store) list(prefix []byte) ([][]byte, error) {
	if s == nil || s.db == nil {
		return nil, errNoStore
	}
	var items [][]byte
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			val, err := item.ValueCopy([]byte{})
			if err != nil {
				return err
			}
			items = append(items, val)
		}
		return nil
	})

	return items, err
}

func (s *store) close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
