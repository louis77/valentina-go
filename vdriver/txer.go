// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vdriver

// Transactions are not supported in Valentina DB.
// TODO: Other engines might support transactions.

type vTx struct {
	conn *vConn
}

func (tx vTx) Commit() error {
	return ErrTxNotImplemented
}

func (tx vTx) Rollback() error {
	return ErrTxNotImplemented
}
