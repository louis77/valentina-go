// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	contentType   = "application/json"
	acceptType    = "application/json"
	defaultVendor = "Valentina"

	kDateFormat    = "kYMD"
	kDateSeparater = "-"
)

var (
	ErrTxNotImplemented = fmt.Errorf("transactions are not supported")
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

// Connection helpers
func (c *vConn) getDatabasePropertyString(name string) (string, error) {
	rows, err := c.QueryContext(context.Background(), "GET PROPERTY "+name+" OF DATABASE", nil)
	if err != nil {
		return "", fmt.Errorf("cannot get database property: %w", err)
	}
	defer rows.Close()

	values := make([]driver.Value, 1)

	err = rows.Next(values)
	if err != nil {
		return "", fmt.Errorf("cannot get database property row: %w", err)
	}

	strval, ok := values[0].(string)
	if !ok {
		return "", fmt.Errorf("database properties is not a string")
	}
	return strval, nil
}

func (c *vConn) setDatabasePropertyString(name string, value driver.Value) error {
	_, err := c.ExecContext(context.Background(), "SET PROPERTY "+name+" OF DATABASE TO ?", []driver.NamedValue{
		{Name: "", Ordinal: 1, Value: value},
	})
	if err != nil {
		return fmt.Errorf("cannot set database properties: %w", err)
	}

	return nil
}

// Connection internals
func (c *vConn) makeRequest(ctx context.Context, method string, resource string, body any) (*http.Response, error) {
	var payload io.ReadCloser
	var bodyBytes []byte // To preserve the encoded body

	if body != nil {
		buf := new(bytes.Buffer)
		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(body); err != nil {
			return nil, fmt.Errorf("json encoding error: %w", err)
		}
		bodyBytes = buf.Bytes()                            // Save the encoded bytes
		payload = io.NopCloser(bytes.NewReader(bodyBytes)) // Use a fresh reader
	}

	endpoint := c.restURL.Scheme + "://" + c.restURL.Host + resource

	req, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", acceptType)
	if c.sessionID != "" {
		req.Header.Set("Cookie", "sessionID="+c.sessionID)
	}

	if bodyBytes != nil {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes)) // Ensure body is reusable
		req.ContentLength = int64(len(bodyBytes))           // Set Content-Length for clarity
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	return resp, nil
}

func readResponseBody[T any](resp *http.Response) (*T, error) {
	defer resp.Body.Close()

	var result T
	enc := json.NewDecoder(resp.Body)
	err := enc.Decode(&result)
	return &result, err
}

func (c *vConn) createSession() error {
	ctx := context.Background()

	password, _ := c.restURL.User.Password()
	hasher := md5.New()
	hasher.Write([]byte(password))
	hashBytes := hasher.Sum(nil)
	hashedPassword := hex.EncodeToString(hashBytes)

	payload := map[string]string{
		"user":     c.restURL.User.Username(),
		"password": hashedPassword,
	}

	resp, err := c.makeRequest(ctx, http.MethodPost, "/rest", payload)
	if err != nil {
		return fmt.Errorf("createSession failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("unexpected http code: %v (unknown error)", resp.StatusCode)
		}
		var verr vError
		if err := json.Unmarshal(msg, &verr); err == nil {
			return fmt.Errorf("valentina error: %s", verr.Error)
		}

		return fmt.Errorf("error: %s", string(msg))
	}

	cookieHeader := resp.Header.Get("Set-Cookie")
	if cookieHeader == "" {
		return fmt.Errorf("missing Set-Cookie header")
	}
	cookieParts := strings.Split(cookieHeader, "=")
	if len(cookieParts) != 2 || cookieParts[0] != "sessionID" {
		return fmt.Errorf("invalid Set-Cookie header")
	}

	sessionID := cookieParts[1]
	if sessionID == "" {
		return fmt.Errorf("invalid Set-Cookie header")
	}
	c.sessionID = sessionID

	return nil
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

type vResult struct {
	affectedRows int64
	lastInsertId int64
}

func (r vResult) LastInsertId() (int64, error) {
	return r.affectedRows, nil
}

func (r vResult) RowsAffected() (int64, error) {
	return r.lastInsertId, nil
}
