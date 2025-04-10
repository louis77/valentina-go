// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver

import (
	"context"
	"database/sql/driver"
	"fmt"
	"net/http"
)

func (c *vConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	// Use Fast SQL
	msg := vFastSQLRequest{
		Vendor:   c.vendor,
		Database: c.database[1:],
		Query:    query,
	}

	if len(args) > 0 {
		msg.Params = make([]any, len(args))
		for i, arg := range args {
			msg.Params[i] = arg.Value
		}
	}

	resource := fmt.Sprintf("/rest/session_%s/sql_fast", c.sessionID)
	resp, err := c.makeRequest(ctx, http.MethodPost, resource, msg)
	if err != nil {
		return nil, fmt.Errorf("makeRequest failed: %w", err)
	}

	response, err := readResponseBody[vFastSQLResponse](resp)
	if err != nil {
		return nil, fmt.Errorf("json decoding failed: %w", err)
	}
	if response.Error != "" {
		// Session expired, tell Go to refresh it
		if resp.StatusCode == http.StatusNotFound && response.Error == "Session does not exist" {
			return nil, driver.ErrBadConn
		}

		return nil, fmt.Errorf("valentina error: %s", response.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	// We can either have a Result_Table or AffectedRows (in case user is not using Execer)
	switch {
	case response.Name == "Result_Table":
		var rows vRows
		rows.columns = response.Fields
		rows.records = response.Records
		rows.pos = 0
		return &rows, nil
	case response.Name == "" && response.AffectedRows > 0:
		// We artificially create a affected_rows row
		var rows vRows
		rows.columns = []string{"affected_rows"}
		rows.records = [][]any{{response.AffectedRows}}
		rows.pos = 0
		return &rows, nil
	}

	// Still here? Then we have an error
	return nil, fmt.Errorf("unexpected response type: %s", response.Name)
}
