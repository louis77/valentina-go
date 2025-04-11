// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver_test

import (
	"database/sql"
	"testing"

	"github.com/louis77/valentina-go/vdriver"
)

func TestConnector(t *testing.T) {
	connector := vdriver.NewConnector(vdriver.VendorValentina, vdriver.Config{
		DB:       "",
		User:     "sa",
		Password: "sa",
		Host:     "localhost",
		Port:     19998,
		UseSSL:   false,
	})

	db := sql.OpenDB(connector)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	}()

	if err := db.Ping(); err != nil {
		t.Fatalf("connector failed to ping: %v", err)
	}
}

func TestConnectorShouldFail(t *testing.T) {
	connector := vdriver.NewConnector(vdriver.VendorValentina, vdriver.Config{
		DB:       "",
		User:     "sa",
		Password: "sa",
		Host:     "somewhere.else",
		Port:     19998,
		UseSSL:   false,
	})

	db := sql.OpenDB(connector)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	}()

	if err := db.Ping(); err == nil {
		t.Fatalf("should have failed")
	}
}
