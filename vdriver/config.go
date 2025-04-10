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
	Vendor   Vendor
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
		Scheme:   scheme,
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:     "/" + cfg.DB,
		RawQuery: fmt.Sprintf("vendor=%s", cfg.Vendor),
	}

	return connURL.String()
}
