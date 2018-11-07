package main

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		fmt.Println("creating table channels...")
		_, err := db.Exec(`
			CREATE TABLE channels(
				id UUID PRIMARY KEY DEFAULT uuid_generate_v1mc(),
				token UUID UNIQUE DEFAULT uuid_generate_v1mc(),
				created_at DATE
			)
		`)
		if err != nil {
			return err
		}

		fmt.Println("creating table lines...")
		_, err = db.Exec(`
			CREATE TABLE lines(
				id SERIAL PRIMARY KEY,
				content BYTEA,
				channel_id UUID REFERENCES channels(id)
			)
		`)
		return err
	}, func(db migrations.DB) error {
		fmt.Println("dropping table lines...")
		if _, err := db.Exec(`DROP TABLE lines`); err != nil {
			return err
		}

		fmt.Println("dropping table channels...")
		_, err := db.Exec(`DROP TABLE channels`)
		return err
	})
}
