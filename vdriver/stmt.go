// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver

import (
	"context"
	"database/sql/driver"
)

type vStmt struct {
	conn  *vConn
	query string
}

func (s vStmt) Close() error {
	return nil
}

func (s vStmt) NumInput() int {
	return -1 // TODO: at the moment we don't know how many placeholders are in the query
}

func (s vStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.conn.ExecContext(context.Background(), s.query, valuesToNamedValues(args))
}

func (s vStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.conn.QueryContext(context.Background(), s.query, valuesToNamedValues(args))
}

func valuesToNamedValues(args []driver.Value) []driver.NamedValue {
	var namedArgs []driver.NamedValue
	for i, arg := range args {
		namedArgs = append(namedArgs, driver.NamedValue{
			Name:    "",
			Ordinal: i,
			Value:   arg,
		})
	}
	return namedArgs
}
