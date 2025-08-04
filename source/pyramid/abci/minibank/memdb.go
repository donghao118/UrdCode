package minibank

import (
	"emulator/utils"
	"emulator/utils/store"
	"fmt"
)

type MemDB interface {
	GetSetMemDB

	GetReadSet() map[string]uint32
	GetWriteSet() map[string]uint32
	CommitWrite(bool) error

	Write() error
}

type GetSetMemDB interface {
	Get(key string) (uint32, error)
	Has(key string) (bool, error)
	Set(key string, value uint32) error
}


type MemoryDB struct {
	db          *store.PrefixStore
	memDB       map[string]uint32
	writeBuffer map[string]uint32
	readBuffer  map[string]uint32
	// dbMutex sync.RWMutex
}

var _ MemDB = (*MemoryDB)(nil)

func NewMemoryDB(db *store.PrefixStore) *MemoryDB {
	return &MemoryDB{
		db:          db,
		memDB:       make(map[string]uint32),
		writeBuffer: map[string]uint32{},
		readBuffer:  map[string]uint32{},
	}
}

func (m *MemoryDB) Get(key string) (uint32, error) {
	if out, ok := m.writeBuffer[key]; ok {
		return out, nil
	}
	if out, ok := m.readBuffer[key]; ok {
		return out, nil
	}
	if out, ok := m.memDB[key]; ok {
		m.readBuffer[key] = out
		return out, nil
	} else if bz, err := m.db.Get([]byte(key)); err != nil {
		m.readBuffer[key] = initBalance
		return initBalance, nil
	} else if len(bz) == 0 {
		m.readBuffer[key] = initBalance
		//return 0, errors.New("Key doesn't exist")
		return initBalance, nil
	} else {
		amount := utils.BytesToUint32(bz)
		m.memDB[key] = amount
		m.readBuffer[key] = amount
		return amount, nil
	}
}

func (m *MemoryDB) Has(key string) (bool, error) {
	if _, ok := m.writeBuffer[key]; ok {
		return true, nil
	}
	if _, ok := m.memDB[key]; ok {

		if m.readBuffer[key] == 0 {
			m.readBuffer[key] = 0
		}
		return true, nil
	} else {
		return m.db.Has([]byte(key))
	}
}

func (m *MemoryDB) Set(key string, value uint32) error {
	m.writeBuffer[key] = value
	return nil
}
func (m *MemoryDB) GetReadSet() map[string]uint32  { return m.readBuffer }
func (m *MemoryDB) GetWriteSet() map[string]uint32 { return m.writeBuffer }
func (m *MemoryDB) CommitWrite(do bool) error {
	if do {
		for key, value := range m.writeBuffer {
			m.memDB[key] = value
		}
	}
	m.writeBuffer = map[string]uint32{}
	m.readBuffer = map[string]uint32{}
	return nil
}
func (m *MemoryDB) Write() error {
	//m.CommitWrite(true)
	batch, err := m.db.NewBatch()
	if err != nil {
		return err
	}
	defer batch.Close()
	for key, valueInt := range m.memDB {
		value := utils.Uint32ToBytes(valueInt)
		err := batch.Set([]byte(key), value)
		if err != nil {
			return err
		}
	}
	if err := batch.WriteSync(); err != nil {
		return err
	}
	m.clearMemory()
	return nil
}
func (m *MemoryDB) clearMemory() { m.memDB = map[string]uint32{} }

type RangeDB struct {
	db        *MemoryDB
	rangeTree *utils.RangeTree
	// dbMutex sync.RWMutex
}

var _ MemDB = (*RangeDB)(nil)

func NewRangeDB(db *MemoryDB, range_tree *utils.RangeTree) *RangeDB {
	return &RangeDB{
		db:        db,
		rangeTree: range_tree,
	}
}

func (m *RangeDB) Get(key string) (uint32, error) {
	if !m.rangeTree.Search(key) {
		return 0, fmt.Errorf("Out of shard range")
	}
	return m.db.Get(key)
}

func (m *RangeDB) Has(key string) (bool, error) {
	if !m.rangeTree.Search(key) {
		return false, fmt.Errorf("Out of shard range")
	}
	return m.db.Has(key)
}

func (m *RangeDB) Set(key string, value uint32) error {
	if !m.rangeTree.Search(key) {
		return fmt.Errorf("Out of shard range")
	}
	return m.db.Set(key, value)
}

func (m *RangeDB) Write() error {
	//m.CommitWrite(true)
	batch, err := m.db.db.NewBatch()
	if err != nil {
		return err
	}
	defer batch.Close()
	for key, valueInt := range m.db.memDB {
		if m.rangeTree.Search(key) {
			value := utils.Uint32ToBytes(valueInt)
			err := batch.Set([]byte(key), value)
			if err != nil {
				return err
			}
		}
	}
	if err := batch.WriteSync(); err != nil {
		return err
	}
	m.db.clearMemory()
	return nil
}

func (m *RangeDB) GetWriteSet() map[string]uint32 {
	return m.db.GetWriteSet()
}
func (m *RangeDB) GetReadSet() map[string]uint32 { return m.db.GetReadSet() }
func (m *RangeDB) CommitWrite(do bool) error     { return m.db.CommitWrite(do) }

type CrossShardDB struct {
	crossShardData map[string]uint32
	db             *MemoryDB
}

var _ MemDB = (*CrossShardDB)(nil)

func NewCrossShardDB(db *MemoryDB, cross_shard_data map[string]uint32) *CrossShardDB {
	return &CrossShardDB{
		crossShardData: cross_shard_data,
		db:             db,
	}
}

func (m *CrossShardDB) Get(key string) (uint32, error) {
	if out, ok := m.db.writeBuffer[key]; ok {
		return out, nil
	}
	if bz, ok := m.crossShardData[key]; !ok {
		return 0, fmt.Errorf("During the transaction, key=" + key + "is read, but the cross-shard block is not provided")
	} else {
		return bz, nil
	}
}
func (m *CrossShardDB) Set(key string, value uint32) error {
	return m.db.Set(key, value)
}
func (m *CrossShardDB) Has(key string) (bool, error) {
	if _, ok := m.db.writeBuffer[key]; ok {
		return true, nil
	}
	_, ok := m.crossShardData[key]
	return ok, nil
}

func (m *CrossShardDB) Write() error {
	return m.db.Write()
}

func (m *CrossShardDB) GetReadSet() map[string]uint32  { return m.crossShardData }
func (m *CrossShardDB) GetWriteSet() map[string]uint32 { return m.db.GetWriteSet() }
func (m *CrossShardDB) CommitWrite(do bool) error {
	return m.db.CommitWrite(do)
}