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

func (c *vConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
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

		// This is a special case for statements that have no rows and no effect, like "SET PROPERTY ..."
		if response.Error == "neither cursor nor affectedRows" {
			return vResult{}, nil
		}

		return nil, fmt.Errorf("valentina error: %s", response.Error)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	result := vResult{
		affectedRows: response.AffectedRows,
		lastInsertId: 0, // TODO: is this supported?
	}

	return &result, nil
}
