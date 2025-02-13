// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"regexp"

	"modernc.org/sqlite"

	"github.com/go-kivik/kivik/v4"
	"github.com/go-kivik/kivik/v4/driver"
	internal "github.com/go-kivik/kivik/v4/int/errors"
)

func init() {
	kivik.Register("sqlite", &drv{})

	if err := sqlite.RegisterCollationUtf8("COUCHDB_UCI", couchdbCmpString); err != nil {
		panic(err)
	}
}

type drv struct{}

var _ driver.Driver = (*drv)(nil)

// NewClient returns a new SQLite client. dsn should be the full path to your
// SQLite database file.
func (drv) NewClient(dsn string, options driver.Options) (driver.Client, error) {
	cn := &connector{dsn: dsn}
	options.Apply(cn)
	db, err := cn.Connect()
	if err != nil {
		return nil, err
	}

	c := &client{
		dsn:    dsn,
		db:     db,
		logger: log.Default(),
	}
	options.Apply(c)

	return c, nil
}

type client struct {
	dsn    string
	db     *sql.DB
	logger *log.Logger
}

var _ driver.Client = (*client)(nil)

const (
	version = "0.0.1"
	vendor  = "Kivik"
)

func (client) Version(context.Context) (*driver.Version, error) {
	return &driver.Version{
		Version: version,
		Vendor:  vendor,
	}, nil
}

func (c *client) AllDBs(ctx context.Context, _ driver.Options) ([]string, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT
			name
		FROM
			sqlite_schema
		WHERE
			type ='table' AND
			name NOT LIKE 'sqlite_%'
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dbs []string
	for rows.Next() {
		var db string
		if err := rows.Scan(&db); err != nil {
			return nil, err
		}
		dbs = append(dbs, db)
	}
	return dbs, rows.Err()
}

func (c *client) DBExists(ctx context.Context, name string, _ driver.Options) (bool, error) {
	var exists bool
	err := c.db.QueryRowContext(ctx, `
		SELECT
			TRUE
		FROM
			sqlite_schema
		WHERE
			type = 'table' AND
			name = ?
		`, name).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return exists, nil
}

var validDBNameRE = regexp.MustCompile(`^[a-z][a-z0-9_$()+/-]*$`)

func (c *client) CreateDB(ctx context.Context, name string, _ driver.Options) error {
	if !validDBNameRE.MatchString(name) {
		return &internal.Error{Status: http.StatusBadRequest, Message: "invalid database name"}
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	d := c.newDB(name)
	for _, query := range schema {
		_, err := tx.ExecContext(ctx, d.query(query))
		if err == nil {
			continue
		}
		if errIsAlreadyExists(err) {
			return &internal.Error{Status: http.StatusPreconditionFailed, Message: "database already exists"}
		}
		return err
	}
	return tx.Commit()
}

func (c *client) DestroyDB(ctx context.Context, name string, _ driver.Options) error {
	if !validDBNameRE.MatchString(name) {
		return &internal.Error{Status: http.StatusBadRequest, Message: "invalid database name"}
	}
	_, err := c.db.ExecContext(ctx, `DROP TABLE "`+name+`"`)
	if err == nil {
		return nil
	}
	if errIsNoSuchTable(err) {
		return &internal.Error{Status: http.StatusNotFound, Message: "database not found"}
	}
	return err
}

func (c *client) DB(name string, _ driver.Options) (driver.DB, error) {
	if !validDBNameRE.MatchString(name) {
		return nil, &internal.Error{Status: http.StatusBadRequest, Message: "invalid database name"}
	}
	return c.newDB(name), nil
}
