package store

import (
	"bytes"
	dbm "emulator/libs/db"
	"fmt"
)

var (
	stateKey   = []byte("0")
	dataKey    = []byte("1")
	specialKey = []byte("2")
)

type ProtoBytesInterface interface {
	ProtoBytes() []byte
	Hash() []byte
}

func toDataKey(bz []byte) []byte {
	return bytes.Join([][]byte{dataKey, bz}, nil)
}
func fromDataKey(bz []byte) []byte {
	return bytes.TrimPrefix(bz, dataKey)
}
func toStateKey(bz []byte) []byte {
	return bytes.Join([][]byte{stateKey, bz}, nil)
}
func toSpecialKey(bz []byte) []byte {
	return bytes.Join([][]byte{specialKey, bz}, nil)
}

type PrefixStore struct {
	Database dbm.DB
}

func NewPrefixStore(name, dir string) *PrefixStore {
	db, err := dbm.NewDB(name, dbm.RocksDBBackend, dir)
	if err != nil {
		panic(err)
	}
	return &PrefixStore{Database: db}
}

func (p *PrefixStore) GetState(key []byte) ([]byte, error) {
	return p.Database.Get(toStateKey(key))
}
func (p *PrefixStore) Get(key []byte) ([]byte, error) {
	return p.Database.Get(toDataKey(key))
}

func (p *PrefixStore) HasState(key []byte) (bool, error) {
	return p.Database.Has(toStateKey(key))
}
func (p *PrefixStore) Has(key []byte) (bool, error) {
	return p.Database.Has(toDataKey(key))
}

func (p *PrefixStore) SetState(key, state []byte) error {
	return p.Database.Set(toStateKey(key), state)
}
func (p *PrefixStore) Set(key, value []byte) error {
	return p.Database.Set(toDataKey(key), value)
}

func (p *PrefixStore) HasSpecial(key []byte) (bool, error) {
	return p.Database.Has(toSpecialKey(key))
}
func (p *PrefixStore) SetSpecial(key, value []byte) error {
	return p.Database.Set(toSpecialKey(key), value)
}
func (p *PrefixStore) GetSpecial(key []byte) ([]byte, error) {
	return p.Database.Get(toSpecialKey(key))
}

func (p *PrefixStore) DeleteState() error {
	return p.Database.Delete(stateKey)
}
func (p *PrefixStore) Delete(key []byte) error {
	return p.Database.Delete(toDataKey(key))
}

func (p *PrefixStore) Iterator(start, end []byte) (*PrefixIterator, error) {
	iter, err := p.Database.Iterator(toDataKey(start), toDataKey(end))
	return &PrefixIterator{iter: iter}, err
}
func (p *PrefixStore) ReverseIterator(start, end []byte) (*PrefixIterator, error) {
	iter, err := p.Database.ReverseIterator(toDataKey(start), toDataKey(end))
	return &PrefixIterator{iter: iter}, err
}
func (p *PrefixStore) Close() error {
	return p.Database.Close()
}
func (p *PrefixStore) NewBatch() (*PrefixBatch, error) {
	batch := p.Database.NewBatch()
	return &PrefixBatch{batch: batch}, nil
}

func (p *PrefixStore) GetBlockByHeight(height int64, chain string) ([]byte, error) {
	key := fmt.Sprintf("%d:%s", height, chain)
	hs, err := p.GetState([]byte(key))
	if err != nil {
		return nil, err
	}
	return p.GetBlockByHash(hs)
}
func (p *PrefixStore) GetBlockByHash(hs []byte) ([]byte, error) {
	return p.Get(hs)
}
func (p *PrefixStore) SetBlockByHeight(height int64, chain string, block ProtoBytesInterface) error {
	hs := block.Hash()
	key := fmt.Sprintf("%d:%s", height, chain)
	if err := p.SetState([]byte(key), hs); err != nil {
		return err
	}
	return p.SetBlockByHash(hs, block)
}
func (p *PrefixStore) SetBlockByHash(hs []byte, block ProtoBytesInterface) error {
	return p.Set(hs, block.ProtoBytes())
}

type PrefixIterator struct {
	iter dbm.Iterator
}

func (pi *PrefixIterator) Domain() ([]byte, []byte) {
	start, end := pi.iter.Domain()
	return fromDataKey(start), fromDataKey(end)
}
func (pi *PrefixIterator) Valid() bool {
	return pi.iter.Valid()
}
func (pi *PrefixIterator) Next()         { pi.iter.Next() }
func (pi *PrefixIterator) Key() []byte   { return fromDataKey(pi.iter.Key()) }
func (pi *PrefixIterator) Value() []byte { return pi.iter.Value() }
func (pi *PrefixIterator) Error() error  { return pi.iter.Error() }
func (pi *PrefixIterator) Close() error  { return pi.iter.Error() }

type PrefixBatch struct {
	batch dbm.Batch
}

func (pi *PrefixBatch) Set(key, value []byte) error {
	return pi.batch.Set(toDataKey(key), value)
}
func (pi *PrefixBatch) Delete(key []byte) error {
	return pi.batch.Delete(toDataKey(key))
}
func (pi *PrefixBatch) Write() error     { return pi.batch.Write() }
func (pi *PrefixBatch) WriteSync() error { return pi.batch.WriteSync() }
func (pi *PrefixBatch) Close() error     { return pi.batch.Close() }
