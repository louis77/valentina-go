// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver_test

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/louis77/valentina-go/vdriver"
)

func Test(t *testing.T) {
	db, err := sql.Open("valentina", "http://sa:sa@localhost:19998/testdb")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	row := db.QueryRow("SELECT now()")

	var now string
	err = row.Scan(&now)
	if err != nil {
		t.Fatalf("failed to scan row: %v", err)
	}

	fmt.Printf("Now() is %s\n", now)
	fmt.Printf("Connections in use: %d\n", db.Stats().InUse)
}
