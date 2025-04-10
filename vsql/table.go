// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vsql

import "database/sql"

type TableMeta struct {
	ID          int
	Name        string
	Encoding    string
	Locale      string
	RecordCount float64
	FieldCount  int
	StorageType string
}

// Tables returns a list of all user tables in the database
func Tables(db *sql.DB) ([]TableMeta, error) {
	rows, err := db.Query(`SELECT 
fld_name, fld_id, fld_encoding, fld_locale, fld_record_count, fld_field_count,
fld_storage_type
FROM (SHOW TABLES) 
WHERE fld_type = 'TABLE' 
AND fld_kind_str = 'USER'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableMeta
	for rows.Next() {
		var table TableMeta
		err := rows.Scan(&table.Name, &table.ID, &table.Encoding, &table.Locale, &table.RecordCount, &table.FieldCount, &table.StorageType)
		if err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}
