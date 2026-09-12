package receipt

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mint/internal/floor"
	"mint/internal/snapshot"
	"mint/internal/statehome"
)

func TestStoreIsImmutableAndValidationDetectsStaleness(t *testing.T) {
	root := repo(t)
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	write(t, filepath.Join(root, "app.txt"), "one\n")
	runGit(t, root, "add", "app.txt")
	source, err := snapshot.Capture(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	result := floor.Result{Pass: true, Clauses: []floor.ClauseResult{{Clause: 1, Name: "completion", Pass: true}}}
	record, err := New(NewOptions{
		Slug: "unit", SpecID: "001", AttemptID: "a1", Terminal: "done-verified", Snapshot: source,
		Result: result, Input: floor.Input{Verdict: map[string]any{"executor": "codex", "vendor": "openai", "model": "gpt", "locality": "remote", "executionRef": "checker"}},
		IssuedAt: time.Unix(1, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	path, err := Store(root, record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Store(root, record); err == nil {
		t.Fatal("second store overwrote immutable receipt")
	}
	loaded, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := Validate(root, loaded); !got.Valid || !got.Current {
		t.Fatalf("fresh validation = %#v", got)
	}
	write(t, filepath.Join(root, "app.txt"), "two\n")
	if got := Validate(root, loaded); !got.Valid || got.Current || got.Reason == "" {
		t.Fatalf("stale validation = %#v", got)
	}
}

func TestOrphanClaimsOnlyReturnsClaimsWithoutReceipts(t *testing.T) {
	root := repo(t)
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	dir := filepath.Join(stateDir(root), "receipts", "unit", "001")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "a1.completed"), "present\n")
	write(t, filepath.Join(dir, "present.json"), "{}\n")
	write(t, filepath.Join(dir, "a2.completed"), "missing\n")
	write(t, filepath.Join(dir, "a3.completed"), "../invalid\n")
	got := OrphanClaims(root)
	if len(got) != 2 || filepath.Base(got[0]) != "a2.completed" || filepath.Base(got[1]) != "a3.completed" {
		t.Fatalf("orphan claims = %v", got)
	}
}

func stateDir(root string) string {
	return statehome.Resolve(root).Dir
}

func repo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	return root
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A receipt issued on a single-vendor box is still a receipt. It just carries
// weaker independence, and a reader has to be able to see that.
func TestVendorDiverseRecordsCheckerIndependence(t *testing.T) {
	root := repo(t)
	t.Setenv("MINT_STATE_HOME", t.TempDir())
	write(t, filepath.Join(root, "app.txt"), "one\n")
	runGit(t, root, "add", "app.txt")
	source, err := snapshot.Capture(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	result := floor.Result{Pass: true, Clauses: []floor.ClauseResult{{Clause: 1, Name: "completion", Pass: true}}}
	build := func(checkerVendor string) Record {
		record, err := New(NewOptions{
			Slug: "unit", SpecID: "001", AttemptID: "a1", Terminal: "done-verified", Snapshot: source,
			Result: result,
			Input: floor.Input{
				Verdict: map[string]any{
					"executor": "claude", "vendor": checkerVendor, "model": "claude",
					"locality": "remote", "executionRef": "checker",
				},
				MakerExecutor: "claude", MakerVendor: "Anthropic", MakerModel: "claude",
				MakerLocality: "remote", MakerExecutionRef: "maker",
			},
			IssuedAt: time.Unix(1, 0),
		})
		if err != nil {
			t.Fatal(err)
		}
		return record
	}
	if got := build("openai").VendorDiverse; got == nil || !*got {
		t.Fatal("a different checker vendor was not recorded as diverse")
	}
	if got := build("anthropic").VendorDiverse; got == nil || *got {
		t.Fatal("a same-vendor checker was recorded as diverse (case-insensitive match failed)")
	}

	// A receipt written before the field existed must not read back as weaker
	// evidence than it holds: both provenances are still in it.
	record := build("openai")
	record.VendorDiverse = nil
	blob, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), "vendorDiverse") {
		t.Fatal("an unrecorded vendorDiverse was serialized as false")
	}
	path := filepath.Join(t.TempDir(), "legacy.json")
	write(t, path, string(blob))
	loaded, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.VendorDiverse == nil || !*loaded.VendorDiverse {
		t.Fatal("a legacy receipt with different vendors read back as not diverse")
	}
}
