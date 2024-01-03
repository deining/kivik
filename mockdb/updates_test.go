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

package mockdb_test

import (
	"context"
	"testing"

	"github.com/go-kivik/kivik/v4/mockdb"
)

func TestDBUpdates(t *testing.T) {
	client, mock, err := mockdb.New()
	if err != nil {
		panic(err)
	}
	mock.ExpectDBUpdates().WillReturn(mockdb.NewDBUpdates().LastSeq("99-last"))

	updates := client.DBUpdates(context.Background())
	for updates.Next() {
		/* .. do nothing .. */
	}
	lastSeq, err := updates.LastSeq()
	if err != nil {
		t.Fatal(err)
	}
	if lastSeq != "99-last" {
		t.Errorf("Unexpected lastSeq: %s", lastSeq)
	}
}
