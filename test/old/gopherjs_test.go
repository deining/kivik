// +build js ignore

package test

import (
	"testing"

	"github.com/gopherjs/gopherjs/js"

	"github.com/flimzy/kivik"
	"github.com/flimzy/kivik/driver/pouchdb"
)

func init() {
	kivik.Register("memdown", &pouchdb.Driver{
		Defaults: map[string]interface{}{
			"db": js.Global.Call("require", "memdown"),
		},
	})
}

func TestPouchLocal(t *testing.T) {
	client, err := kivik.New("memdown", "")
	if err != nil {
		t.Errorf("Failed to connect to PouchDB/memdown driver: %s", err)
		return
	}
	clients := &Clients{
		Admin: client,
	}
	RunSubtests(clients, true, SuitePouchLocal, t)
}

func TestPouchRemote(t *testing.T) {
	doTest(SuitePouchRemote, "KIVIK_TEST_DSN_COUCH16", true, t)
}
