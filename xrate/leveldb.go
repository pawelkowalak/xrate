package xrate

import (
	"fmt"

	"encoding/json"
	"github.com/syndtr/goleveldb/leveldb"
)

// NewDatabase returns new levelDB object.
func NewDatabase(db *leveldb.DB) Database {
	return &levelDB{db: db}
}

type levelDB struct {
	db *leveldb.DB
}

func (d levelDB) Get(key []byte) (*FixerRates, error) {
	buf, err := d.db.Get(key, nil)
	if err != nil {
		switch err {
		case leveldb.ErrNotFound:
			return nil, err
		default:
			return nil, fmt.Errorf("can't fetch key %s: %v", string(key), err)
		}
	}

	fr := new(FixerRates)
	if err := json.Unmarshal(buf, fr); err != nil {
		return nil, fmt.Errorf("can't unmarshal value for key %s: %v", string(key), err)
	}
	return fr, nil
}

func (d levelDB) Set(key, value []byte) error {
	return d.db.Put(key, value, nil)
}
