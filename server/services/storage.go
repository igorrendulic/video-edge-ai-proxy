// Copyright 2020 Wearless Tech Inc All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	badger "github.com/dgraph-io/badger/v2"
)

const (
	TablePrefixRTSP = "/rtsp/"
)

// Storage - main storage functions (Get, Put, Del, List)
type Storage struct {
	db *badger.DB
}

func NewStorage(db *badger.DB) *Storage {
	return &Storage{
		db: db,
	}
}

func (s *Storage) Put(prefix, key string, value []byte) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(prefix+key), value)
		return err
	})
	return err
}

func (s *Storage) Get(prefix, key string) ([]byte, error) {
	var valCopy []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(prefix + key))
		if err != nil {
			return err
		}
		valCopy, err = item.ValueCopy(nil)
		return err
	})
	return valCopy, err
}

func (s *Storage) Del(prefix, key string) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(prefix + key))
		return err
	})
	return err
}

func (s *Storage) List(prefix string) (map[string][]byte, error) {

	results := make(map[string][]byte, 0)
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 128
		it := txn.NewIterator(opts)
		defer it.Close()
		pfix := []byte(prefix)
		for it.Seek(pfix); it.ValidForPrefix(pfix); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				results[string(k)] = v
				return nil
			})
			if err != nil {
				g.Log.Error("failed to iterate in db", err)
				return err
			}
		}
		return nil
	})
	return results, err
}
