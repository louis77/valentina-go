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

type Connector struct {
	vendor Vendor
	config Config
}

func NewConnector(vendor Vendor, config Config) driver.Connector {
	return Connector{
		vendor: vendor,
		config: config,
	}
}

func (c Connector) Connect(ctx context.Context) (driver.Conn, error) {
	hc := http.Client{
		// Make sure redirects are not followed
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	conn := vConn{
		httpClient: &hc,
		restURL:    c.config.makeURL(),
		vendor:     string(c.vendor),
	}

	if err := conn.createSession(); err != nil {
		return nil, fmt.Errorf("cannot create session: %w", err)
	}

	// Set the date format for this connection. This is required for the time.Time type.
	// Both proerties are only visible to the current connection and not persisted in the database,
	// see https://valentina-db.com/docs/dokuwiki/v15/doku.php?id=valentina:vcomponents:vkernel:database:datetime_format&s[]=kymd
	err := conn.setDatabasePropertyString("DateTimeFormat", kDateFormat)
	if err != nil {
		return nil, fmt.Errorf("cannot set database DateTimeFormat property: %w", err)
	}

	err = conn.setDatabasePropertyString("DateSeparator", kDateSeparater)
	if err != nil {
		return nil, fmt.Errorf("cannot set database DateSeparator property: %w", err)
	}

	return &conn, nil
}

func (c Connector) Driver() driver.Driver {
	return vDriver{
		Vendor: c.vendor,
	}
}

func (c Connector) Close() error {
	return nil
}

