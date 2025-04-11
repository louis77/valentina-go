// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver

import (
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type vConn struct {
	httpClient *http.Client
	restURL    *url.URL
	sessionID  string
	database   string
	vendor     string
}

func (c *vConn) Prepare(query string) (driver.Stmt, error) {
	return &vStmt{
		query: query,
		conn:  c,
	}, nil
}

// Close removes the REST session from the server
func (c *vConn) Close() error {
	ctx := context.Background()
	resp, err := c.makeRequest(ctx, http.MethodDelete, "/rest/session_"+c.sessionID, nil)
	if err != nil {
		return fmt.Errorf("makeRequest failed: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		// Check if it has a body
		if resp.ContentLength == 0 {
			return fmt.Errorf("unexpected status code: %v", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err == nil {
			return fmt.Errorf("error: %s", string(body))
		}
	}

	c.sessionID = ""
	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *vConn) Begin() (driver.Tx, error) {
	return vTx{
		conn: c,
	}, nil
}

func (c *vConn) Ping(ctx context.Context) error {
	rows, err := c.QueryContext(ctx, "SELECT version()", nil)
	if err != nil {
		return err
	}
	defer rows.Close()

	var version string
	err = rows.Next([]driver.Value{&version})
	if err != nil {
		return err
	}

	if version == "" {
		return fmt.Errorf("valentina error: version() returned an empty string")
	}
	fmt.Printf("Valentina server version: %s\n", version)
	return nil
}
