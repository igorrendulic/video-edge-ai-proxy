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
	"os"
	"strconv"
	"testing"

	g "github.com/chryscloud/video-edge-ai-proxy/globals"
	badger "github.com/dgraph-io/badger/v2"
)

var (
	testDBPath = "/tmp/chrystest"
)

func setupDB() (*badger.DB, error) {
	if _, err := os.Stat(testDBPath); os.IsNotExist(err) {
		// path/to/whatever does not exist
		errDir := os.Mkdir(testDBPath, 0755)
		if errDir != nil {
			g.Log.Error("failed to create directiory for DB", testDBPath, errDir)
			return nil, errDir
		}
	} else {
		err := os.RemoveAll(testDBPath)
		if err != nil {
			return nil, err
		}
	}
	db, err := badger.Open(badger.DefaultOptions(testDBPath))
	if err != nil {
		g.Log.Error("faile to open database", err)
		return nil, err
	}
	return db, nil
}

func TestStorage(t *testing.T) {
	db, err := setupDB()
	if err != nil {
		t.Fatal(err)
	}
	prefix := "/test/"
	testValue := "this is test"
	s := NewStorage(db)
	pErr := s.Put(prefix, "test", []byte(testValue))
	if pErr != nil {
		t.Fatal(pErr)
	}
	val, gErr := s.Get(prefix, "test")
	if gErr != nil {
		t.Fatal(gErr)
	}
	if string(val) != testValue {
		t.Fatal("test value not equal")
	}
}

func TestPrefixScan(t *testing.T) {
	db, err := setupDB()
	if err != nil {
		t.Fatal(err)
	}
	prefix := "/testprefix/"
	testValue := "this is test"
	s := NewStorage(db)
	for x := 1; x <= 10; x++ {
		val := testValue + "_" + strconv.Itoa(x)
		key := prefix + strconv.Itoa(x)
		s.Put(prefix, key, []byte(val))
	}
	res, err := s.List(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 10 {
		t.Fatal("expected 10 results")
	}

}
