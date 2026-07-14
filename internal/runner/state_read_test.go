package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

// The state file is already durable when the executor exits, so an existing file
// is read on the FIRST attempt with no fixed delay — this guards against a
// reintroduced multi-second sleep on the hot path.
func TestReadStateWithRetry_ImmediateFileNoDelay(t *testing.T) {
	RegisterTestingT(t)
	dir := t.TempDir()
	Expect(os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{"status":0}`), 0600)).To(Succeed())
	root, err := os.OpenRoot(dir)
	Expect(err).To(BeNil())
	defer func() { _ = root.Close() }()

	start := time.Now()
	b, err := readStateWithRetry(root, "state.json")
	elapsed := time.Since(start)

	Expect(err).To(BeNil())
	Expect(string(b)).To(Equal(`{"status":0}`))
	Expect(elapsed).To(BeNumerically("<", 100*time.Millisecond)) // no blind wait
}

// A file that appears slightly late is still read within the bounded window.
func TestReadStateWithRetry_FileAppearsLate(t *testing.T) {
	RegisterTestingT(t)
	origN, origI := stateReadMaxAttempts, stateReadRetryInterval
	stateReadMaxAttempts, stateReadRetryInterval = 100, 5*time.Millisecond
	defer func() { stateReadMaxAttempts, stateReadRetryInterval = origN, origI }()

	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	Expect(err).To(BeNil())
	defer func() { _ = root.Close() }()

	go func() {
		time.Sleep(25 * time.Millisecond)
		_ = os.WriteFile(filepath.Join(dir, "state.json"), []byte(`{"status":0}`), 0600)
	}()

	b, err := readStateWithRetry(root, "state.json")
	Expect(err).To(BeNil())
	Expect(len(b)).To(BeNumerically(">", 0))
}

// A file that never appears returns an error after the bounded attempts (no
// infinite loop / no 5s wait).
func TestReadStateWithRetry_MissingFileErrors(t *testing.T) {
	RegisterTestingT(t)
	origN, origI := stateReadMaxAttempts, stateReadRetryInterval
	stateReadMaxAttempts, stateReadRetryInterval = 3, time.Millisecond
	defer func() { stateReadMaxAttempts, stateReadRetryInterval = origN, origI }()

	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	Expect(err).To(BeNil())
	defer func() { _ = root.Close() }()

	_, err = readStateWithRetry(root, "missing.json")
	Expect(err).To(Not(BeNil()))
}
