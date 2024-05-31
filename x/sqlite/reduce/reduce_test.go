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

package reduce

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gitlab.com/flimzy/testy"

	"github.com/go-kivik/kivik/v4"
)

func reduceCount(_ [][2]interface{}, values []interface{}, rereduce bool) ([]interface{}, error) {
	if !rereduce {
		return []any{len(values)}, nil
	}
	var total float64
	for _, value := range values {
		total += value.(float64)
	}
	return []any{total}, nil
}

func TestReduce(t *testing.T) {
	type test struct {
		input      []Row
		fn         Func
		want       []Row
		wantErr    string
		wantStatus int
	}

	tests := testy.NewTable()
	tests.Add("no inputs", test{})
	tests.Add("count single row", test{
		input: []Row{
			{ID: "1", Key: "foo", Value: nil},
		},
		fn: reduceCount,
		want: []Row{
			{Value: 1},
		},
	})
	tests.Add("count two rows", test{
		input: []Row{
			{ID: "1", Key: "foo", Value: nil},
			{ID: "2", Key: "foo", Value: nil},
		},
		fn: reduceCount,
		want: []Row{
			{Value: 2},
		},
	})

	tests.Run(t, func(t *testing.T, tt test) {
		got, err := Reduce(tt.input, tt.fn)
		if !testy.ErrorMatches(tt.wantErr, err) {
			t.Errorf("Unexpected error: %v", err)
		}
		if status := kivik.HTTPStatus(err); status != tt.wantStatus {
			t.Errorf("Unexpected status code: %d", status)
		}
		if err != nil {
			return
		}
		if d := cmp.Diff(tt.want, got); d != "" {
			t.Errorf("Unexpected output (-want +got):\n%s", d)
		}
	})
}
