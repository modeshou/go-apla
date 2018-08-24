//go:generate sh -c "mockery -inpkg -name Database -print > file.tmp && mv file.tmp database_mock.go"
//go:generate sh -c "mockery -inpkg -name Transaction -print > file.tmp && mv file.tmp transaction_mock.go"

package kv

import (
	"database/sql/driver"
	"io"
)

type Database interface {
	io.Closer

	// Starting read/read-write transaction
	Begin(writable bool) Transaction
}

type Transaction interface {
	Set(key, val string) error
	Update(key, val string) (string, error)
	Delete(key string) error
	Get(key string) (string, error)

	AddIndex(index *IndexAdapter)
	Ascend(index string, iterator func(key, value string) bool) error

	driver.Tx
}