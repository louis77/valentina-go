// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver_test

import (
	"database/sql"
	"testing"

	_ "github.com/louis77/valentina-go/vdriver"
)

func TestSimpleQuery(t *testing.T) {
	db, err := sql.Open("valentina", "http://sa:sa@localhost:19998/?vendor=Valentina")
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
