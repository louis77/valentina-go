// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	resp, err := c.makeRequest(ctx, http.MethodDelete, "/rest/session_id", nil)
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

	values := make([]driver.Value, 1)
	err = rows.Next(values)
	if err != nil {
		return err
	}

	version, ok := values[0].(string)
	if !ok {
		return fmt.Errorf("valentina error: version() returned invalid type")
	}

	if version == "" {
		return fmt.Errorf("valentina error: version() returned an empty string")
	}

	return nil
}

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
