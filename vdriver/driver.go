// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	contentType          = "application/json"
	acceptType           = "application/json"
	defaultVendor Vendor = VendorValentina

	kDateFormat    = "kYMD"
	kDateSeparater = "-"
)

var (
	ErrTxNotImplemented = fmt.Errorf("transactions are not supported")
	ErrNotSupported     = fmt.Errorf("this feature is not supported")
)

func init() {
	sql.Register("valentina", vDriver{Vendor: VendorValentina})
	sql.Register("vsqlite", vDriver{Vendor: VendorSQLite})
	sql.Register("vduckdb", vDriver{Vendor: VendorDuckDB})
}

// Driver

type vDriver struct {
	Vendor Vendor
}

func (d vDriver) Open(restURL string) (driver.Conn, error) {
	hc := http.Client{
		// Make sure redirects are not followed
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	parsedURL, err := url.Parse(restURL)
	if err != nil {
		return nil, fmt.Errorf("url malformed: %w", err)
	}

	database := parsedURL.Path
	if database != "" {
		database = strings.TrimPrefix(database, "/")
	}

	conn := vConn{
		httpClient: &hc,
		restURL:    parsedURL,
		database:   database,
		vendor:     string(d.Vendor),
	}

	if err := conn.createSession(); err != nil {
		return nil, fmt.Errorf("cannot create session: %w", err)
	}

	// Set the date format for this connection. This is required for the time.Time type.
	// Both proerties are only visible to the current connection and not persisted in the database,
	// see https://valentina-db.com/docs/dokuwiki/v15/doku.php?id=valentina:vcomponents:vkernel:database:datetime_format&s[]=kymd
	err = conn.setDatabasePropertyString("DateTimeFormat", kDateFormat)
	if err != nil {
		return nil, fmt.Errorf("cannot set database DateTimeFormat property: %w", err)
	}

	err = conn.setDatabasePropertyString("DateSeparator", kDateSeparater)
	if err != nil {
		return nil, fmt.Errorf("cannot set database DateSeparator property: %w", err)
	}

	return &conn, nil
}

type vError struct {
	Error string
}

func readResponseBody[T any](resp *http.Response) (*T, error) {
	defer resp.Body.Close()

	var result T
	enc := json.NewDecoder(resp.Body)
	err := enc.Decode(&result)
	return &result, err
}

type vFastSQLRequest struct {
	Vendor   string `json:"vendor"`
	Database string `json:"database"`
	Query    string `json:"Query"`
	Params   []any  `json:"Params,omitzero"`
}

type vFastSQLResponse struct {
	AffectedRows int64

	Name    string   `json:"name"`
	Fields  []string `json:"fields"`
	Records [][]any  `json:"records"`

	vError
}
