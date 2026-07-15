package execstate

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"mint/internal/unitstore"
)

func TestAttemptProvenanceIsImmutableAndWritesAreSerialized(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	if err := unitstore.Ensure(root); err != nil {
		t.Fatal(err)
	}
	spec := unitstore.SpecPath(root, "unit", "001")
	if err := os.MkdirAll(filepath.Dir(spec), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec, []byte("<task/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	maker := &Maker{Executor: "codex", Vendor: "openai", Model: "gpt", Locality: "remote", ExecutionRef: "maker-1"}
	if _, err := Init(root, "unit", "001", "a1", maker); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, "unit", "001", "a1", maker); err == nil {
		t.Fatal("second init overwrote immutable maker")
	}

	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := RecordGate(root, "unit", "001", fmt.Sprintf("g%02d", i), "pass", "a1")
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	state, ok := Read(root, "unit", "001", "a1")
	if !ok || len(state.Gates) != 20 || state.Maker.Executor != "codex" {
		t.Fatalf("state=%#v", state)
	}
	if info, err := os.Stat(Path(root, "unit", "001", "a1")); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%v err=%v", info.Mode(), err)
	}
}

func TestGenericProvenanceValidation(t *testing.T) {
	valid := Provenance{Executor: "custom", Vendor: "vendor", Model: "model", Locality: "local", ExecutionRef: "run-1"}
	if _, err := ValidateProvenance(valid); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []Provenance{{}, {Executor: "x", Vendor: "v", Model: "m", Locality: "claimed", ExecutionRef: "r"}, {Executor: "x", Vendor: "v", Model: "m", Locality: "local", ExecutionRef: "r", ObservedBy: "driver"}} {
		if _, err := ValidateProvenance(bad); err == nil {
			t.Fatalf("accepted %#v", bad)
		}
	}
}
