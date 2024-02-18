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

var schema = []string{
	// revs
	`CREATE TABLE %[2]q (
		id TEXT NOT NULL,
		rev INTEGER NOT NULL,
		rev_id TEXT NOT NULL,
		parent_rev INTEGER,
		parent_rev_id TEXT,
		FOREIGN KEY (id, parent_rev, parent_rev_id) REFERENCES %[2]q (id, rev, rev_id) ON DELETE CASCADE,
		UNIQUE(id, rev, rev_id)
	)`,
	`CREATE INDEX idx_parent ON %[2]q (id, parent_rev, parent_rev_id)`,
	// the main db table
	`CREATE TABLE %[1]q (
		seq INTEGER PRIMARY KEY,
		id TEXT NOT NULL,
		rev INTEGER NOT NULL,
		rev_id TEXT NOT NULL,
		doc BLOB NOT NULL,
		deleted BOOLEAN NOT NULL DEFAULT FALSE,
		FOREIGN KEY (id, rev, rev_id) REFERENCES %[2]q (id, rev, rev_id) ON DELETE CASCADE,
		UNIQUE(id, rev, rev_id)
	)`,
	// attachments
	`CREATE TABLE %[3]q (
		id TEXT NOT NULL,
		rev INTEGER NOT NULL,
		rev_id TEXT NOT NULL,
		filename TEXT NOT NULL,
		content_type TEXT NOT NULL,
		length INTEGER NOT NULL,
		digest TEXT NOT NULL,
		data BLOB NOT NULL,
		FOREIGN KEY (id, rev, rev_id) REFERENCES %[2]q (id, rev, rev_id) ON DELETE CASCADE,
		UNIQUE(id, rev, rev_id, filename)
	)
	`,
}
