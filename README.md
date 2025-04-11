# Valentina Go Driver

[![Go](https://github.com/louis77/valentina-go/actions/workflows/go.yml/badge.svg)](https://github.com/louis77/valentina-go/actions/workflows/go.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/louis77/valentina-go)](https://goreportcard.com/report/github.com/louis77/valentina-go)

This is a non-offical Go driver for Valentina DB based on Valentina REST API. It implements the [database/sql](https://pkg.go.dev/database/sql) package.

Tested with [Valentina DB](https://valentina-db.com) 15.1.2. It should work with versions >= v.15.0.1, which introduced the REST API.

Why use Valentina's REST API and not the native binary protocol? The later is proprietary, closed-source and would require CGo and linking with the official Valentina C SDK. The maintainer decided to use the dependency-free REST API instead.

## Installation

```bash
go get github.com/louis77/valentina-go
```

## Usage

This driver is using Valentina REST API. You need to activate it in Valentina server by setting an appropriate port in the [`vserver.ini`](https://valentina-db.com/docs/dokuwiki/v15/doku.php?id=valentina:products:vserver:manual:ini_file) file:

```ini
[REST]
PORT_REST = 19998
PORT_REST_SSL=19999
```

After that, you can use the driver like any other `sql/database` driver, see [Limitations](#limitations) below.

```go
package main

import (
	"database/sql"

	_ "github.com/louis77/valentina-go/vdriver"
)

func main() {
	// Valentina DB
	db, err := sql.Open("valentina", "http://sa:sa@localhost:19998")
	// Valentina SQLite
	// db, err := sql.Open("vsqlite", "http://sa:sa@localhost:19998")
	// Valentina SQLite
	// db, err := sql.Open("vduckdb", "http://sa:sa@localhost:19998")

	row := db.QueryRow("SELECT now(), :1 as a_number", 69)

	var now string
	var anumber int
	err = row.Scan(&now, &anumber)
	if err != nil {
		t.Fatalf("failed to scan row: %v", err)
	}

	// Use data
}
```

You can also use the `vdriver.Config` struct to configure the connection:

```go
cfg := vdriver.Config{
	DB:       "testdb",
	User:     "sa",
	Password: "sa",
	Host:     "localhost",
	Port:     19998,
	UseSSL:   false,
}

db, err := sql.Open("valentina", cfg.FormatDSN())
```

Valentina Server supports three different engines. Use the `driverName` to indicate which engine you want use:

- Valentina DB: `valentina`
- Valentina SQLite: `vsqlite`
- Valentina DuckDB: `vduckdb`

### Use the CLI

This package contains a small CLI tool to connect to Valentina DB and execute SQL queries. It can be used as follows:

```bash
$ go install ./cmd/vsql

# Show help
$ vsql -h
```

You can also use the `.verbose` command to see the results in a tabular format.

### Connection Parameters

The driver accepts the following parameters:

- `vendor`: the vendor name (default: `Valentina`, others: `SQLite`, `DuckDB`).

## Special Types

### DateTime

The `vsql` package provides a special type `Time` that can be used to scan Valentina time values. It implements the `Scanner` and `Valuer`. The driver sets the appropriate `DateFormat` and `DateSeparator` properties in the database for each new connection.

```go
row := db.QueryRow("SELECT now()")
var now vsql.Time
err = row.Scan(&now)
```

### Arrays

Valentina supports the `ARRAY` type which is a fixed-size array of a specific underlying type. You can scan an array by using `[]any` as the destination type.

## Notes about Valentina SQL

Placeholders for parameters are prefixed with a colon (`:`) and a number, starting from 1. This way, the same parameter can be used multiple times in the query:

```sql
SELECT :1 + :2
```

Alternatively, you can use MySQL-style `?` placeholders:

```sql
SELECT ? + ?
```

The driver will automatically convert the parameters to the right type.

## Limitations

- Valentina does not support transactions
- Valentina does not support implicit LastInsertId() when using Exec(). You need to fetch it with `SELECT Last_RecID()`
- Prepared statements work, the REST API doesn't supporting caching statements, so each execution of a prepared statement will send the full query text to the server
- Expired REST sessions are automatically refreshed, queries will not fail because of an expired session
- If your license allows only a limited number of REST connections, don't forget to set the maximum open connections, i.e.: `db.SetMaxOpenConns(3)`

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This project is licensed under the BSD 3-Clause License - see the [LICENSE](LICENSE) file for details.  "Valentina", "Valentina Server", "Valentina Studio" are trademarks of [Paradigma Software, Inc](https://www.valentina-db.com). This project is a voluntary effort and neither paid for nor affiliated with Paradigma Software.
