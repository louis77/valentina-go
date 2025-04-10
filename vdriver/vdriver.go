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
)

var (
	ErrSessionTimeout = fmt.Errorf("REST session timed out")
)

func init() {
	sql.Register("valentina", vDriver{})
}

// Driver

type vDriver struct{}

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
	if database == "" {
		return nil, fmt.Errorf("missing database name")
	}

	params := parsedURL.Query()
	vendor := params.Get("vendor")
	if vendor == "" {
		vendor = defaultVendor
	}

	conn := vConn{
		httpClient: &hc,
		restURL:    parsedURL,
		database:   database,
		vendor:     vendor,
	}

	if err := conn.createSession(); err != nil {
		return nil, fmt.Errorf("cannot create session: %w", err)
	}

	return &conn, nil
}

type vError struct {
	Error string
}

// Connection

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

// Close idle HTTP connection to the server. Otherwise it's doing nothing.
func (c *vConn) Close() error {
	ctx := context.Background()
	resp, err := c.makeRequest(ctx, http.MethodDelete, "/rest/"+c.sessionID, nil)
	if err != nil {
		return fmt.Errorf("makeRequest failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Check if it has a body
		if resp.ContentLength == 0 {
			return fmt.Errorf("unexpected status code: %v", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			return fmt.Errorf("error: %s", string(body))
		}
	}

	c.httpClient.CloseIdleConnections()
	return nil
}

func (c *vConn) Begin() (driver.Tx, error) {
	return vTx{}, nil
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

	req, err := http.NewRequestWithContext(ctx, method, c.restURL.Scheme+"://"+c.restURL.Host+":"+resource, payload)
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
	if len(cookieParts) != 2 {
		return fmt.Errorf("invalid Set-Cookie header")
	}
	if cookieParts[0] != "sessionID" {
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
	Vendor   string         `json:"vendor"`
	Database string         `json:"database"`
	Query    string         `json:"Query"`
	Params   map[string]any `json:"Params"`
}

type vFastSQLResponse struct {
	AffectedRows int64 // Should this be a distinct struct?

	Name    string   `json:"name"`
	Fields  []string `json:"fields"`
	Records [][]any  `json:"records"`

	vError
}

func (c *vConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	// Use Fast SQL
	msg := vFastSQLRequest{
		Vendor:   c.vendor,
		Database: c.database[1:],
		Query:    query,
		Params:   make(map[string]any),
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
		// We artificialle create a affected_rows row
		var rows vRows
		rows.columns = []string{"affected_rows"}
		rows.records = [][]any{{response.AffectedRows}}
		rows.pos = 0
		return &rows, nil
	}

	// Still here? Then we have an error
	return nil, fmt.Errorf("unexpected response type: %s", response.Name)
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

func (c *vConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	// Use Fast SQL
	msg := vFastSQLRequest{
		Vendor:   c.vendor,
		Database: c.database[1:],
		Query:    query,
		Params:   make(map[string]any),
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

// Pinger

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

// Statement

type vStmt struct {
	conn  *vConn
	query string
}

func (s vStmt) Close() error {
	return nil
}

func (s vStmt) NumInput() int {
	return 0
}

func (s vStmt) Exec(args []driver.Value) (driver.Result, error) {
	return vResult{}, nil
}

func (s vStmt) Query(args []driver.Value) (driver.Rows, error) {
	var namedArgs []driver.NamedValue
	for i, arg := range args {
		namedArgs = append(namedArgs, driver.NamedValue{
			Name:    "",
			Ordinal: i,
			Value:   arg,
		})
	}
	return s.conn.QueryContext(context.Background(), s.query, namedArgs)
}

// Tx

type vTx struct {
}

func (tx vTx) Commit() error {
	return nil
}

func (tx vTx) Rollback() error {
	return nil
}

// Row

type vRows struct {
	columns []string
	records [][]any
	pos     int
}

func (rows *vRows) Columns() []string {
	return rows.columns
}

func (rows *vRows) Close() error { return nil }

func (rows *vRows) Next(dest []driver.Value) error {
	rows.pos++
	if rows.pos > len(rows.records) {
		return io.EOF
	}

	row := rows.records[rows.pos-1]
	for idx, v := range row {
		dest[idx] = v
	}

	return nil
}
