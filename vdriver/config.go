// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver

import (
	"fmt"
	"net/url"
)

type Vendor string

const (
	VendorValentina Vendor = "Valentina"
	VendorSQLite    Vendor = "SQLite"
	VendorDuckDB    Vendor = "DuckDB"
)

type Config struct {
	DB       string
	User     string
	Password string
	Host     string
	Port     int
	UseSSL   bool
}

func (cfg Config) FormatDSN() string {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}

	connURL := url.URL{
		Scheme: scheme,
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   "/" + cfg.DB,
	}

	return connURL.String()
}
