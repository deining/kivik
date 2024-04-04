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
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"

	"github.com/go-kivik/kivik/v4/driver"
)

type changes struct {
	rows    *sql.Rows
	lastSeq string
	etag    string
}

var _ driver.Changes = &changes{}

func (c *changes) Next(change *driver.Change) error {
	if !c.rows.Next() {
		if err := c.rows.Err(); err != nil {
			return err
		}
		return io.EOF
	}
	var rev string
	if err := c.rows.Scan(&change.ID, &change.Seq, &change.Deleted, &rev); err != nil {
		return err
	}
	change.Changes = driver.ChangedRevs{rev}
	c.lastSeq = change.Seq
	return nil
}

func (c *changes) Close() error {
	return c.rows.Close()
}

func (c *changes) LastSeq() string {
	// Columns returns an error if the rows are closed, so we can use that to
	// determine if we've actually read the last sequence id.
	if _, err := c.rows.Columns(); err == nil {
		return ""
	}
	return c.lastSeq
}

func (c *changes) Pending() int64 {
	return 0
}

func (c *changes) ETag() string {
	return c.etag
}

func (d *db) Changes(ctx context.Context, _ driver.Options) (driver.Changes, error) {
	query := d.query(`
		WITH results AS (
			SELECT
				id,
				seq,
				deleted,
				rev,
				rev_id
			FROM test
			ORDER BY seq
		)
		SELECT
			NULL AS id,
			NULL AS seq,
			NULL AS deleted,
			COUNT(*) || '.' || COALESCE(MIN(seq),0) || '.' || COALESCE(MAX(seq),0) AS rev
		FROM results

		UNION ALL

		SELECT
			id,
			seq,
			deleted,
			rev || '-' || rev_id AS rev
		FROM results
	`)
	rows, err := d.db.QueryContext(ctx, query) //nolint:rowserrcheck // Err checked in Next
	if err != nil {
		return nil, err
	}

	// The first row is used to calculate the ETag; it's done as part of the
	// same query, even though it's a bit ugly, to ensure it's all in the same
	// implicit transaction.
	if !rows.Next() {
		// should never happen
		return nil, errors.New("no rows returned")
	}
	var discard *string
	var summary string
	if err := rows.Scan(&discard, &discard, &discard, &summary); err != nil {
		return nil, err
	}

	h := md5.New()
	_, _ = h.Write([]byte(summary))

	return &changes{
		rows: rows,
		etag: hex.EncodeToString(h.Sum(nil)),
	}, nil
}
