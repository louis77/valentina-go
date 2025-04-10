// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vsql

import (
	"database/sql/driver"
	"time"
)

type Time struct {
	time.Time
}

func (t Time) Value() (driver.Value, error) {
	// This is the default date/time format in Valentina.
	// However, this can be changed in the database.
	// TODO Check if we need to do anything about it.

	return t.Format("01/02/2006 15:04:05:00"), nil
}
