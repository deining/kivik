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

//go:build !js
// +build !js

package sqlite

import (
	"context"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gitlab.com/flimzy/testy"

	"github.com/go-kivik/kivik/v4"
	"github.com/go-kivik/kivik/v4/driver"
	"github.com/go-kivik/kivik/v4/internal/mock"
)

type rowResult struct {
	ID    string
	Rev   string
	Key   string
	Value string
	Doc   string
	Error string
}

func TestDBAllDocs(t *testing.T) {
	t.Parallel()
	type test struct {
		db         driver.DB
		options    driver.Options
		want       []rowResult
		wantStatus int
		wantErr    string
	}
	tests := testy.NewTable()
	tests.Add("no docs in db", test{
		want: nil,
	})
	tests.Add("single doc", func(t *testing.T) interface{} {
		db := newDB(t)
		rev := db.tPut("foo", map[string]string{"cat": "meow"})

		return test{
			db: db,
			want: []rowResult{
				{
					ID:    "foo",
					Key:   "foo",
					Value: `{"value":{"rev":"` + rev + `"}}`,
				},
			},
		}
	})
	tests.Add("include_docs=true", func(t *testing.T) interface{} {
		db := newDB(t)
		rev := db.tPut("foo", map[string]string{"cat": "meow"})

		return test{
			db:      db,
			options: kivik.Param("include_docs", true),
			want: []rowResult{
				{
					ID:    "foo",
					Key:   "foo",
					Value: `{"value":{"rev":"` + rev + `"}}`,
					Doc:   `{"_id":"foo","_rev":"` + rev + `","cat":"meow"}`,
				},
			},
		}
	})
	tests.Add("single doc multiple revisions", func(t *testing.T) interface{} {
		db := newDB(t)
		rev := db.tPut("foo", map[string]string{"cat": "meow"})
		rev2 := db.tPut("foo", map[string]string{"cat": "purr"}, kivik.Rev(rev))

		return test{
			db: db,
			want: []rowResult{
				{
					ID:    "foo",
					Key:   "foo",
					Value: `{"value":{"rev":"` + rev2 + `"}}`,
				},
			},
		}
	})
	tests.Add("conflicting document, select winning rev", func(t *testing.T) interface{} {
		db := newDB(t)
		_ = db.tPut("foo", map[string]string{
			"cat":  "meow",
			"_rev": "1-xxx",
		}, kivik.Param("new_edits", false))
		_ = db.tPut("foo", map[string]string{
			"cat":  "purr",
			"_rev": "1-aaa",
		}, kivik.Param("new_edits", false))

		return test{
			db: db,
			want: []rowResult{
				{
					ID:    "foo",
					Key:   "foo",
					Value: `{"value":{"rev":"1-xxx"}}`,
				},
			},
		}
	})
	tests.Add("deleted doc", func(t *testing.T) interface{} {
		db := newDB(t)
		rev := db.tPut("foo", map[string]string{"cat": "meow"})
		_ = db.tDelete("foo", kivik.Rev(rev))

		return test{
			db:   db,
			want: nil,
		}
	})
	tests.Add("select lower revision number when higher rev in winning branch has been deleted", func(t *testing.T) interface{} {
		db := newDB(t)
		_ = db.tPut("foo", map[string]string{
			"cat":  "meow",
			"_rev": "1-xxx",
		}, kivik.Param("new_edits", false))
		_ = db.tPut("foo", map[string]string{
			"cat":  "purr",
			"_rev": "1-aaa",
		}, kivik.Param("new_edits", false))
		_ = db.tDelete("foo", kivik.Rev("1-aaa"))

		return test{
			db: db,
			want: []rowResult{
				{
					ID:    "foo",
					Key:   "foo",
					Value: `{"value":{"rev":"1-xxx"}}`,
				},
			},
		}
	})
	tests.Add("conflicts=true", func(t *testing.T) interface{} {
		db := newDB(t)
		_ = db.tPut("foo", map[string]string{
			"cat":  "meow",
			"_rev": "1-xxx",
		}, kivik.Param("new_edits", false))
		_ = db.tPut("foo", map[string]string{
			"cat":  "purr",
			"_rev": "1-aaa",
		}, kivik.Param("new_edits", false))

		return test{
			db: db,
			options: kivik.Params(map[string]interface{}{
				"conflicts":    true,
				"include_docs": true,
			}),
			want: []rowResult{
				{
					ID:    "foo",
					Key:   "foo",
					Value: `{"value":{"rev":"1-xxx"}}`,
					Doc:   `{"_id":"foo","_rev":"1-xxx","cat":"meow","_conflicts":["1-aaa"]}`,
				},
			},
		}
	})
	tests.Add("conflicts=true ignored without include_docs", func(t *testing.T) interface{} {
		db := newDB(t)
		_ = db.tPut("foo", map[string]string{
			"cat":  "meow",
			"_rev": "1-xxx",
		}, kivik.Param("new_edits", false))
		_ = db.tPut("foo", map[string]string{
			"cat":  "purr",
			"_rev": "1-aaa",
		}, kivik.Param("new_edits", false))

		return test{
			db: db,
			options: kivik.Params(map[string]interface{}{
				"conflicts": true,
			}),
			want: []rowResult{
				{
					ID:    "foo",
					Key:   "foo",
					Value: `{"value":{"rev":"1-xxx"}}`,
				},
			},
		}
	})
	tests.Add("default sorting", func(t *testing.T) interface{} {
		db := newDB(t)
		rev1 := db.tPut("cat", map[string]string{
			"cat": "meow",
		})
		rev2 := db.tPut("dog", map[string]string{
			"dog": "woof",
		})
		rev3 := db.tPut("cow", map[string]string{
			"cow": "moo",
		})

		return test{
			db: db,
			want: []rowResult{
				{
					ID:    "cat",
					Key:   "cat",
					Value: `{"value":{"rev":"` + rev1 + `"}}`,
				},
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
				{
					ID:    "dog",
					Key:   "dog",
					Value: `{"value":{"rev":"` + rev2 + `"}}`,
				},
			},
		}
	})
	tests.Add("descending=true", func(t *testing.T) interface{} {
		db := newDB(t)
		rev1 := db.tPut("cat", map[string]string{
			"cat": "meow",
		})
		rev2 := db.tPut("dog", map[string]string{
			"dog": "woof",
		})
		rev3 := db.tPut("cow", map[string]string{
			"cow": "moo",
		})

		return test{
			db:      db,
			options: kivik.Param("descending", true),
			want: []rowResult{
				{
					ID:    "dog",
					Key:   "dog",
					Value: `{"value":{"rev":"` + rev2 + `"}}`,
				},
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
				{
					ID:    "cat",
					Key:   "cat",
					Value: `{"value":{"rev":"` + rev1 + `"}}`,
				},
			},
		}
	})
	tests.Add("endkey", func(t *testing.T) interface{} {
		db := newDB(t)
		rev1 := db.tPut("cat", map[string]string{
			"cat": "meow",
		})
		_ = db.tPut("dog", map[string]string{
			"dog": "woof",
		})
		rev3 := db.tPut("cow", map[string]string{
			"cow": "moo",
		})

		return test{
			db:      db,
			options: kivik.Param("endkey", "cow"),
			want: []rowResult{
				{
					ID:    "cat",
					Key:   "cat",
					Value: `{"value":{"rev":"` + rev1 + `"}}`,
				},
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
			},
		}
	})
	tests.Add("descending=true, endkey", func(t *testing.T) interface{} {
		db := newDB(t)
		_ = db.tPut("cat", map[string]string{
			"cat": "meow",
		})
		rev2 := db.tPut("dog", map[string]string{
			"dog": "woof",
		})
		rev3 := db.tPut("cow", map[string]string{
			"cow": "moo",
		})

		return test{
			db: db,
			options: kivik.Params(map[string]interface{}{
				"endkey":     "cow",
				"descending": true,
			}),
			want: []rowResult{
				{
					ID:    "dog",
					Key:   "dog",
					Value: `{"value":{"rev":"` + rev2 + `"}}`,
				},
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
			},
		}
	})
	tests.Add("end_key", func(t *testing.T) interface{} {
		db := newDB(t)
		rev1 := db.tPut("cat", map[string]string{
			"cat": "meow",
		})
		_ = db.tPut("dog", map[string]string{
			"dog": "woof",
		})
		rev3 := db.tPut("cow", map[string]string{
			"cow": "moo",
		})

		return test{
			db:      db,
			options: kivik.Param("end_key", "cow"),
			want: []rowResult{
				{
					ID:    "cat",
					Key:   "cat",
					Value: `{"value":{"rev":"` + rev1 + `"}}`,
				},
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
			},
		}
	})
	tests.Add("endkey, inclusive_end=false", func(t *testing.T) interface{} {
		db := newDB(t)
		rev1 := db.tPut("cat", map[string]string{
			"cat": "meow",
		})
		_ = db.tPut("dog", map[string]string{
			"dog": "woof",
		})
		_ = db.tPut("cow", map[string]string{
			"cow": "moo",
		})

		return test{
			db: db,
			options: kivik.Params(map[string]interface{}{
				"endkey":        "cow",
				"inclusive_end": false,
			}),
			want: []rowResult{
				{
					ID:    "cat",
					Key:   "cat",
					Value: `{"value":{"rev":"` + rev1 + `"}}`,
				},
			},
		}
	})
	tests.Add("startkey", func(t *testing.T) interface{} {
		db := newDB(t)
		_ = db.tPut("cat", map[string]string{
			"cat": "meow",
		})
		rev2 := db.tPut("dog", map[string]string{
			"dog": "woof",
		})
		rev3 := db.tPut("cow", map[string]string{
			"cow": "moo",
		})

		return test{
			db:      db,
			options: kivik.Param("startkey", "cow"),
			want: []rowResult{
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
				{
					ID:    "dog",
					Key:   "dog",
					Value: `{"value":{"rev":"` + rev2 + `"}}`,
				},
			},
		}
	})
	tests.Add("start_key", func(t *testing.T) interface{} {
		db := newDB(t)
		_ = db.tPut("cat", map[string]string{
			"cat": "meow",
		})
		rev2 := db.tPut("dog", map[string]string{
			"dog": "woof",
		})
		rev3 := db.tPut("cow", map[string]string{
			"cow": "moo",
		})

		return test{
			db:      db,
			options: kivik.Param("start_key", "cow"),
			want: []rowResult{
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
				{
					ID:    "dog",
					Key:   "dog",
					Value: `{"value":{"rev":"` + rev2 + `"}}`,
				},
			},
		}
	})
	tests.Add("startkey, descending", func(t *testing.T) interface{} {
		db := newDB(t)
		rev1 := db.tPut("cat", map[string]string{
			"cat": "meow",
		})
		_ = db.tPut("dog", map[string]string{
			"dog": "woof",
		})
		rev3 := db.tPut("cow", map[string]string{
			"cow": "moo",
		})

		return test{
			db: db,
			options: kivik.Params(map[string]interface{}{
				"startkey":   "cow",
				"descending": true,
			}),
			want: []rowResult{
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
				{
					ID:    "cat",
					Key:   "cat",
					Value: `{"value":{"rev":"` + rev1 + `"}}`,
				},
			},
		}
	})
	tests.Add("limit=2 returns first two documents only", func(t *testing.T) interface{} {
		d := newDB(t)
		rev1 := d.tPut("cat", map[string]string{"cat": "meow"})
		_ = d.tPut("dog", map[string]string{"dog": "woof"})
		rev3 := d.tPut("cow", map[string]string{"cow": "moo"})

		return test{
			db:      d,
			options: kivik.Param("limit", 2),
			want: []rowResult{
				{
					ID:    "cat",
					Key:   "cat",
					Value: `{"value":{"rev":"` + rev1 + `"}}`,
				},
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
			},
		}
	})
	tests.Add("skip=2 skips first two documents", func(t *testing.T) interface{} {
		d := newDB(t)
		_ = d.tPut("cat", map[string]string{"cat": "meow"})
		rev2 := d.tPut("dog", map[string]string{"dog": "woof"})
		_ = d.tPut("cow", map[string]string{"cow": "moo"})

		return test{
			db:      d,
			options: kivik.Param("skip", 2),
			want: []rowResult{
				{
					ID:    "dog",
					Key:   "dog",
					Value: `{"value":{"rev":"` + rev2 + `"}}`,
				},
			},
		}
	})
	tests.Add("limit=1,skip=1 skips 1, limits 1", func(t *testing.T) interface{} {
		d := newDB(t)
		_ = d.tPut("cat", map[string]string{"cat": "meow"})
		_ = d.tPut("dog", map[string]string{"dog": "woof"})
		rev3 := d.tPut("cow", map[string]string{"cow": "moo"})

		return test{
			db:      d,
			options: kivik.Params(map[string]interface{}{"limit": 1, "skip": 1}),
			want: []rowResult{
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
			},
		}
	})
	tests.Add("local docs excluded", func(t *testing.T) interface{} {
		d := newDB(t)
		rev := d.tPut("cat", map[string]string{"cat": "meow"})
		_ = d.tPut("_local/dog", map[string]string{"dog": "woof"})
		rev3 := d.tPut("cow", map[string]string{"cow": "moo"})

		return test{
			db: d,
			want: []rowResult{
				{
					ID:    "cat",
					Key:   "cat",
					Value: `{"value":{"rev":"` + rev + `"}}`,
				},
				{
					ID:    "cow",
					Key:   "cow",
					Value: `{"value":{"rev":"` + rev3 + `"}}`,
				},
			},
		}
	})

	/*
		TODO:
		- Options:
			- endkey_docid
			- end_key_doc_id
			- group
			- group_level
			- include_docs
			- attachments
			- att_encoding_infio
			- key
			- keys
			- reduce
			- sorted
			- stable
			- statle
			- startkey_docid
			- start_key_doc_id
			- update
			- update_seq
		- AllDocs() called for DB that doesn't exit
		- UpdateSeq() called on rows
		- Offset() called on rows
		- TotalRows() called on rows
	*/

	tests.Run(t, func(t *testing.T, tt test) {
		t.Parallel()
		db := tt.db
		if db == nil {
			db = newDB(t)
		}
		opts := tt.options
		if opts == nil {
			opts = mock.NilOption
		}
		rows, err := db.AllDocs(context.Background(), opts)
		if !testy.ErrorMatches(tt.wantErr, err) {
			t.Errorf("Unexpected error: %s", err)
		}
		if status := kivik.HTTPStatus(err); status != tt.wantStatus {
			t.Errorf("Unexpected status: %d", status)
		}
		if err != nil {
			return
		}

		checkRows(t, rows, tt.want)
	})
}

func checkRows(t *testing.T, rows driver.Rows, want []rowResult) {
	t.Helper()

	// iterate over rows
	var got []rowResult

loop:
	for {
		row := driver.Row{}
		err := rows.Next(&row)
		switch err {
		case io.EOF:
			break loop
		case driver.EOQ:
			continue
		case nil:
			// continue
		default:
			t.Fatalf("Next() returned error: %s", err)
		}
		var errMsg string
		if row.Error != nil {
			errMsg = row.Error.Error()
		}
		var value, doc []byte
		if row.Value != nil {
			value, err = io.ReadAll(row.Value)
			if err != nil {
				t.Fatal(err)
			}
		}

		if row.Doc != nil {
			doc, err = io.ReadAll(row.Doc)
			if err != nil {
				t.Fatal(err)
			}
		}
		got = append(got, rowResult{
			ID:    row.ID,
			Rev:   row.Rev,
			Key:   string(row.Key),
			Value: string(value),
			Doc:   string(doc),
			Error: errMsg,
		})
	}
	if d := cmp.Diff(want, got); d != "" {
		t.Errorf("Unexpected rows:\n%s", d)
	}
}
