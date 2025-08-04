package minibank

import (
	"emulator/utils"
	"emulator/utils/store"
	"fmt"
)

type DB interface {
	Get(string) (uint32, byte, error)
	Set(string, uint32, byte) error

	LoadData(string, uint32)
	Clear()

	WLock(string) error
	WUnlock(string) error
}

func isRLock(locked byte) bool { return locked == RLockedIdentifier }
func isWLock(locked byte) bool { return locked == WLockedIdentifier }
func isFree(locked byte) bool  { return locked == FreeIdentifier }

type AppDB struct {
	db        *store.PrefixStore
	rangelist *utils.RangeList

	retainData map[string]uint32
}

var _ DB = (*AppDB)(nil)

func NewDB(db *store.PrefixStore, rangeList *utils.RangeList) DB {
	return &AppDB{
		db:         db,
		rangelist:  rangeList,
		retainData: make(map[string]uint32),
	}
}

func (app *AppDB) Get(key string) (uint32, byte, error) {
	if app.rangelist.Search(key) {
		if bz, err := app.db.Get([]byte(key)); err != nil || len(bz) == 0 {
			if err := app.Set(key, initBalance, FreeIdentifier); err != nil {
				return 0, '0', err
			}
			return initBalance, FreeIdentifier, nil
		} else {
			return UnmarshalValue(bz)
		}
	} else {
		if v, ok := app.retainData[key]; ok {
			return v, FreeIdentifier, nil
		} else {
			return 0, FreeIdentifier, fmt.Errorf("key does not exists")
		}
	}
}

func (app *AppDB) Set(key string, money uint32, locked byte) error {
	if !app.rangelist.Search(key) {
		return nil
	}
	value, err := MarshalValue(money, locked)
	if err != nil {
		return err
	}
	if err := app.db.Set([]byte(key), value); err != nil {
		return err
	}
	return nil
}

func (app *AppDB) LoadData(key string, value uint32) {
	app.retainData[key] = value
}
func (app *AppDB) Clear() {
	app.retainData = make(map[string]uint32)
}

func (app *AppDB) RLock(key string) error {
	if bz, err := app.db.Get([]byte(key)); err != nil || len(bz) == 0 {
		return fmt.Errorf("key does not exist")
	} else if out := SetValueRLock(bz); out == nil {
		return fmt.Errorf("Unknown error")
	} else if err := app.db.Set([]byte(key), out); err != nil {
		return err
	}
	return nil
}
func (app *AppDB) WLock(key string) error {
	if bz, err := app.db.Get([]byte(key)); err != nil || len(bz) == 0 {
		return fmt.Errorf("key does not exist")
	} else if out := SetValueWLock(bz); out == nil {
		return fmt.Errorf("Unknown error")
	} else if err := app.db.Set([]byte(key), out); err != nil {
		return err
	}
	return nil
}
func (app *AppDB) WUnlock(key string) error {
	if bz, err := app.db.Get([]byte(key)); err != nil || len(bz) == 0 {
		return fmt.Errorf("key does not exist")
	} else if out := SetValueUnlock(bz); out == nil {
		return fmt.Errorf("Unknown error")
	} else if err := app.db.Set([]byte(key), out); err != nil {
		return err
	}
	return nil
}
