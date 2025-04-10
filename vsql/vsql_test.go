// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vsql

import (
	"database/sql"
	"testing"

	_ "github.com/louis77/valentina-go/vdriver"
)

func TestTables(t *testing.T) {
	db, err := sql.Open("valentina", "http://sa:sa@localhost:19998/testdb?vendor=Valentina")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	tables, err := Tables(db)
	if err != nil {
		t.Fatalf("failed to get tables: %v", err)
	}

	if len(tables) == 0 {
		t.Fatal("no tables found")
	}
}
