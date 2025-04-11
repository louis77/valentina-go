// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/louis77/valentina-go/vdriver"
	_ "github.com/louis77/valentina-go/vdriver"
)

func TestSimpleQuery(t *testing.T) {
	db, err := sql.Open("valentina", "http://sa:sa@localhost:19998")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	}()

	row := db.QueryRow("SELECT now(), :1 as a_number", 69)

	var now string
	var anumber int
	err = row.Scan(&now, &anumber)
	if err != nil {
		t.Fatalf("failed to scan row: %v", err)
	}

	if anumber != 69 {
		t.Fatalf("anumber is %d, expected 69", anumber)
	}
}

func TestPing(t *testing.T) {
	db, err := sql.Open("valentina", "http://sa:sa@localhost:19998/?vendor=Valentina")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	}()

	err = db.Ping()
	if err != nil {
		t.Fatalf("failed to ping: %v", err)
	}
}

func TestJSONQuery(t *testing.T) {
	db, err := sql.Open("valentina", "http://sa:sa@localhost:19998/testdb?vendor=Valentina")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close database: %v", err)
		}
	}()

	row := db.QueryRow("SELECT metainfo from testtab")

	var now map[string]interface{}
	err = row.Scan(&now)
	fmt.Println(now)
	if err != nil {
		t.Fatalf("failed to scan row: %v", err)
	}
}

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
	if err := db.Ping(); err != nil {
		t.Fatalf("connector failed to ping: %v", err)
	}
}
