package app

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var updateFixtures = flag.Bool("update-fixtures", false, "rewrite the frontend test fixtures")

// The frontend UI tests run against a fake backend. Its schema is generated
// here from the real backend talking to the built-in test server, so the
// fake sends exactly what the app would. Regenerate with:
//
//	go test -tags webkit2_41 ./internal/app -run TestFrontendFixtures -update-fixtures
func TestFrontendFixtures(t *testing.T) {
	a, rec := newConnectedAPI(t)
	waitFor(t, "services", func() bool {
		for _, e := range rec.all(eventServicesSelectChanged) {
			if len(e.data) > 0 {
				return true
			}
		}
		return false
	})
	var services servicesSelect
	for _, e := range rec.all(eventServicesSelectChanged) {
		if len(e.data) > 0 {
			services = e.data[0].(servicesSelect)
		}
	}

	methods := map[string]interface{}{}
	for _, s := range services {
		for _, m := range s.Methods {
			before := rec.count(eventMethodInputChanged)
			if err := a.SelectMethod(m.FullName, "", nil); err != nil {
				t.Fatalf("SelectMethod(%s): %v", m.FullName, err)
			}
			evs := rec.all(eventMethodInputChanged)
			if len(evs) != before+1 || len(evs[before].data) == 0 {
				t.Fatalf("SelectMethod(%s) did not emit the method input", m.FullName)
			}
			methods[m.FullName] = evs[before].data[0]
		}
	}

	got, err := json.MarshalIndent(map[string]interface{}{
		"services": services,
		"methods":  methods,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	path := filepath.Join("..", "..", "frontend", "tests", "fixtures", "schema.json")
	if *updateFixtures {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (generate it with -update-fixtures)", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is out of date; regenerate it with: go test -tags webkit2_41 ./internal/app -run TestFrontendFixtures -update-fixtures", path)
	}
}
