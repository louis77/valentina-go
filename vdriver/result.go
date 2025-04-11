// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver

type vResult struct {
	affectedRows int64
	lastInsertId int64
}

func (r vResult) LastInsertId() (int64, error) {
	return 0, ErrNotSupported
}

func (r vResult) RowsAffected() (int64, error) {
	return r.affectedRows, nil
}
