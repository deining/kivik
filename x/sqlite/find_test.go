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
	"net/http"
	"testing"

	"gitlab.com/flimzy/testy"

	"github.com/go-kivik/kivik/v4"
	"github.com/go-kivik/kivik/v4/driver"
	"github.com/go-kivik/kivik/v4/int/mock"
)

func TestFind(t *testing.T) {
	t.Parallel()
	type test struct {
		db         *testDB
		query      any
		options    driver.Options
		want       []rowResult
		wantStatus int
		wantErr    string
	}

	tests := testy.NewTable()
	tests.Add("no docs in db", test{
		want: nil,
	})
	tests.Add("query is invalid json", test{
		query:      "invalid json",
		wantStatus: http.StatusBadRequest,
		wantErr:    "invalid character 'i' looking for beginning of value",
	})
	tests.Add("field equality", func(t *testing.T) interface{} {
		d := newDB(t)
		rev := d.tPut("foo", map[string]string{"foo": "bar"})
		_ = d.tPut("bar", map[string]string{"bar": "baz"})

		return test{
			db: d,
			query: map[string]interface{}{
				"foo": "bar",
			},
			want: []rowResult{
				{ID: "foo", Doc: `{"_id":"foo","_rev":"` + rev + `","foo":"bar"}`},
			},
		}
	})

	/*
		TODO:
		- Include _design_docs in results?
		- Include _local_docs in results?
		- limit
		- skip
		- fields
		- use_index
		- bookmark
		- execution_stats
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
		rows, err := db.Find(context.Background(), tt.query, opts)
		if !testy.ErrorMatchesRE(tt.wantErr, err) {
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