package registry_test

import (
	"encoding/json"
	"testing"

	"math/rand"

	"strconv"

	"time"

	"fmt"

	"github.com/GenesisKernel/go-genesis/packages/registry"
	"github.com/GenesisKernel/go-genesis/packages/storage/kv"
	"github.com/GenesisKernel/go-genesis/packages/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/yddmat/memdb"
)

type testModel struct {
	Id     int
	Field  string
	Field2 []byte
}

func newKvDB() (kv.Database, error) {
	db, err := memdb.OpenDB("", false)
	if err != nil {
		return nil, err
	}

	return &kv.DatabaseAdapter{Database: *db}, nil
}

func TestMetadataTx_RW(t *testing.T) {
	cases := []struct {
		testname string

		registry types.Registry
		pkValue  string
		value    interface{}

		expJson string
		err     bool
	}{
		{
			testname: "insert-good",
			registry: types.Registry{
				Name:      "key",
				Ecosystem: &types.Ecosystem{ID: 1},
			},
			pkValue: "1",
			value: testModel{
				Id:     1,
				Field:  "testfield",
				Field2: make([]byte, 10),
			},

			err: false,
		},

		{
			testname: "insert-bad-1",
			registry: types.Registry{
				Name:      "key",
				Ecosystem: &types.Ecosystem{ID: 1},
			},
			pkValue: "1",
			value:   make(chan int),

			err: true,
		},
	}

	for _, c := range cases {
		db, err := newKvDB()
		require.Nil(t, err)

		reg := registry.NewMetadataStorage(db)
		metadataTx := reg.Begin()
		metadataTx.SetBlockHash([]byte("123"))
		metadataTx.SetTxHash([]byte("321"))
		require.Nil(t, err, c.testname)

		err = metadataTx.Insert(&c.registry, c.pkValue, c.value)
		if c.err {
			assert.Error(t, err)
			continue
		}

		assert.Nil(t, err)

		saved := testModel{}
		err = metadataTx.Get(&c.registry, c.pkValue, &saved)
		require.Nil(t, err)

		assert.Equal(t, c.value, saved, c.testname)
	}
}

func TestMetadataTx_benchmark(t *testing.T) {
	db, err := newKvDB()
	require.Nil(t, err)

	storage := registry.NewMetadataStorage(db)
	metadataTx := storage.Begin()

	type key struct {
		ID        int64
		PublicKey []byte
		Amount    int64
		Deleted   bool
		Blocked   bool
	}

	reg := types.Registry{
		Name:      "key",
		Ecosystem: &types.Ecosystem{ID: 1},
	}

	insertStart := time.Now()
	for i := 0; i < 10000; i++ {
		id := rand.Int63()
		err := metadataTx.Insert(
			&reg,
			strconv.FormatInt(id, 10),
			key{
				ID:        id,
				PublicKey: make([]byte, 64),
				Amount:    rand.Int63(),
			},
		)

		if err != nil {
			metadataTx.Commit()
			metadataTx = storage.Begin()
			err = nil
		}

		require.Nil(t, err)
	}

	metadataTx.AddIndex(&kv.IndexAdapter{*memdb.NewIndex("test", "*", func(a, b string) bool {
		return gjson.Get(a, "amount").Less(gjson.Get(b, "amount"), false)
	})})

	require.Nil(t, metadataTx.Commit())
	fmt.Println("Inserted 10.000 keys in", time.Since(insertStart).Seconds())

	readonlyTx := storage.Reader()
	walkingStart := time.Now()
	var topAmount int64
	require.Nil(t, readonlyTx.Walk(&reg, "test", func(jsonRow string) bool {
		k := key{}
		require.Nil(t, json.Unmarshal([]byte(jsonRow), &k))
		if topAmount < k.Amount {
			topAmount = k.Amount
		}
		return true
	}))
	fmt.Println("Finded top amount of 10.000 keys", "in", time.Since(walkingStart))

	secondWriting := time.Now()
	writeTx := storage.Begin()
	writeTx.SetBlockHash([]byte("123"))
	writeTx.SetTxHash([]byte("321"))
	// Insert 10 more values
	for i := -10; i < 0; i++ {
		id := rand.Int63()
		err := writeTx.Insert(
			&reg,
			strconv.FormatInt(id, 10),
			key{
				ID:        id,
				PublicKey: make([]byte, 64),
				Amount:    rand.Int63(),
			},
		)

		require.Nil(t, err)
	}
	require.Nil(t, writeTx.Commit())
	fmt.Println("Inserted 10 keys to 10.000 in", time.Since(secondWriting))
}