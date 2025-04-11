// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vsql

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/louis77/valentina-go/vdriver"
)

func TestTables(t *testing.T) {
	db, err := sql.Open("valentina", "http://sa:sa@localhost:19998/?vendor=Valentina")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS testdb")
	db.Close() // Close the connection to the server, so we can create a new one
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	db2, err := sql.Open("valentina", "http://sa:sa@localhost:19998/testdb?vendor=Valentina")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db2.Close()

	_, err = db2.Exec("CREATE TABLE IF NOT EXISTS testtable (id INT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	tables, err := Tables(db2)
	if err != nil {
		t.Fatalf("failed to get tables: %v", err)
	}

	if len(tables) == 0 {
		t.Fatal("no tables found")
	}
}

func TestTimeScan(t *testing.T) {
	db, err := sql.Open("valentina", "http://sa:sa@localhost:19998/?vendor=Valentina")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	row := db.QueryRow("SELECT NOW()")

	var now Time
	err = row.Scan(&now)
	if err != nil {
		t.Fatalf("failed to scan: %v", err)
	}
}

func TestTimeScan2(t *testing.T) {
	var ourTime Time
	if err := ourTime.Scan("2025-01-02 17:56:39:400"); err != nil {
		t.Fatalf("failed to scan: %v", err)
	}

	if ourTime.Time.Compare(time.Date(2025, 1, 2, 17, 56, 39, 400*1000000, time.UTC)) != 0 {
		t.Fatalf("time is not correct: %v", ourTime.Time.String())
	}
}
