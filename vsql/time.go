// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vsql

import (
	"database/sql/driver"
	"fmt"
	"time"
)

const (
	vTimeFormat = "2006-01-02 15:04:05.000" // Yes, we can't use : as the MS separator, see Scan()
)

type Time struct {
	time.Time
}

func (t Time) Value() (driver.Value, error) {
	// This is the default date/time format in Valentina.
	// However, this can be changed in the database.
	// TODO Check if we need to do anything about it.

	value := t.Format(vTimeFormat)
	value = value[:19] + string(".") + value[20:]
	return value, nil
}

func (t *Time) Scan(value any) error {
	if value == nil {
		*t = Time{}
		return nil
	}
	// Valentina returns the time in the format "2006-01-02 15:04:05:000"
	// We can't use : as the MS separator, so we have to parse it manually

	switch value := value.(type) {
	case string:
		// Replace the MS separator : with a .
		value = value[:19] + string(".") + value[20:]

		tt, err := time.Parse(vTimeFormat, value[:len(vTimeFormat)])
		if err != nil {
			return err
		}
		*t = Time{tt}
		return nil
	case time.Time:
		*t = Time{value}
		return nil
	default:
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type *Time", value)
	}
}
