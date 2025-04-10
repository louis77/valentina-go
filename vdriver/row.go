// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver

import (
	"database/sql/driver"
	"io"
)

type vRows struct {
	columns []string
	records [][]any
	pos     int
}

func (rows *vRows) Columns() []string {
	return rows.columns
}

func (rows *vRows) Close() error { return nil }

func (rows *vRows) Next(dest []driver.Value) error {
	rows.pos++
	if rows.pos > len(rows.records) {
		return io.EOF
	}

	row := rows.records[rows.pos-1]
	for idx, v := range row {
		dest[idx] = v
	}

	return nil
}
