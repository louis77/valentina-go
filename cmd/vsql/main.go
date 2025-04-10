// Copyright 2025 Louis Brauer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/louis77/valentina-go/vdriver"
)

var db *sql.DB

func exec(line string, verbose bool) {
	rows, err := db.Query(line)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	columns, err := rows.Columns()
	if err != nil {
		fmt.Println("columns error", err)
		return
	}

	values := make([][]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}

	if verbose {
		for _, column := range columns {
			fmt.Printf("%v\t", column)
		}
		fmt.Println()
	}

	for rows.Next() {
		err := rows.Scan(pointers...)
		if err != nil {
			panic(err)
		}
		for _, value := range values {
			if value == nil {
				fmt.Printf("%v\t", "NULL")
				continue
			}
			fmt.Printf("%v\t", value)
		}
		fmt.Println()
	}
}

func main() {
	fVendor := flag.String("vendor", string(vdriver.VendorValentina), "The vendor name, default 'Valentina'")
	fDB := flag.String("db", "", "The database name, required")
	fUser := flag.String("u", "sa", "The user name, default 'sa'")
	fPassword := flag.String("p", "sa", "The password, default 'sa'")
	fHost := flag.String("host", "localhost", "The host name, default 'localhost'")
	fPort := flag.Int("port", 0, "The port number, required")
	fSSL := flag.Bool("ssl", false, "Use SSL")
	fHelp := flag.Bool("h", false, "Print this help")
	flag.Parse()

	if *fHelp || *fDB == "" || *fPort == 0 {
		flag.PrintDefaults()
		return
	}

	cfg := vdriver.Config{
		Vendor:   vdriver.Vendor(*fVendor),
		DB:       *fDB,
		User:     *fUser,
		Password: *fPassword,
		Host:     *fHost,
		Port:     *fPort,
		UseSSL:   *fSSL,
	}

	var err error
	db, err = sql.Open("valentina", cfg.FormatDSN())
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "closing database:", err)
		} else {
			fmt.Println("session closed")
		}
	}()

	// if err := db.Ping(); err != nil {
	// 	panic(err)
	// }

	verbose := false
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Press CTRL-D to exit")
repl:
	for {
		fmt.Printf("%s > ", *fDB)
		scanOk := scanner.Scan()

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
			return
		} else if !scanOk {
			fmt.Println()
			return
		}

		line := scanner.Text()

		line = strings.TrimSpace(line)
		switch line {
		case "":
			continue
		case ".verbose":
			verbose = !verbose
			fmt.Println("verbose:", verbose)
		case ".quit":
			break repl
		default:
			exec(line, verbose)
		}
	}
}
