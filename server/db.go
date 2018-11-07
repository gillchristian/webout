package main

import (
	"os"

	"github.com/gillchristian/webout/types"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
)

var opt = pg.Options{
	User:     os.Getenv("POSTGRES_USER"),
	Password: os.Getenv("POSTGRES_PASS"),
	Database: os.Getenv("POSTGRES_DB"),
	Network:  "tcp",
	Addr:     os.Getenv("POSTGRES_ADDR"),
}

func init() {
	if opt.User == "" {
		opt.User = "webout"
	}
	if opt.Password == "" {
		opt.Password = "webout"
	}
	if opt.Database == "" {
		opt.Database = "webout"
	}
	if opt.Addr == "" {
		opt.Addr = "db:5432"
	}
}

func connect() *pg.DB {
	return pg.Connect(&opt)
}

func createSchema(db *pg.DB) error {
	models := []interface{}{
		(*types.Channel)(nil),
	}

	for _, model := range models {
		err := db.CreateTable(model, &orm.CreateTableOptions{
			Temp: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
