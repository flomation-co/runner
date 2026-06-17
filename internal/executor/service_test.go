package executor

import (
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// TestDrainPipeReadsPastOverLongLine is the deterministic correctness guard:
// a line far larger than the old 1MB bufio.Scanner ceiling must be retained up
// to the cap AND the reader must keep going and deliver the following lines.
// The old Scanner-based reader returned ErrTooLong and stopped here, which is
// what silently stalled the pipe and hung the executor.
func TestDrainPipeReadsPastOverLongLine(t *testing.T) {
	RegisterTestingT(t)

	huge := strings.Repeat("A", 5<<20) // 5MB, well over the old 1MB ceiling
	input := huge + "\n" + "__NODE__:after\n" + "tail-no-newline"

	var lines []string
	drainPipe(strings.NewReader(input), maxStoredLineBytes, func(line string) {
		lines = append(lines, line)
	})

	Expect(lines).To(HaveLen(3))

	// Over-long line retained only up to the cap, with the marker appended.
	Expect(len(lines[0])).To(BeNumerically("<=", maxStoredLineBytes+len("… [runner-truncated]")))
	Expect(lines[0]).To(HaveSuffix("… [runner-truncated]"))

	// The line AFTER the over-long one was still read — the regression.
	Expect(lines[1]).To(Equal("__NODE__:after"))

	// A trailing line without a final newline is still delivered.
	Expect(lines[2]).To(Equal("tail-no-newline"))
}

// TestDrainPipeDoesNotDeadlockOnLargeWrite proves the executor (writer) can
// never block: a real os.Pipe is written with a line larger than any OS pipe
// buffer, and the drainer must consume it and finish.
func TestDrainPipeDoesNotDeadlockOnLargeWrite(t *testing.T) {
	RegisterTestingT(t)

	r, w, err := os.Pipe()
	Expect(err).ToNot(HaveOccurred())

	huge := strings.Repeat("A", 5<<20)
	go func() {
		_, _ = io.WriteString(w, huge+"\n")
		_, _ = io.WriteString(w, "__NODE__:after\n")
		_ = w.Close()
	}()

	lines := make(chan []string, 1)
	go func() {
		var collected []string
		drainPipe(r, maxStoredLineBytes, func(line string) {
			collected = append(collected, line)
		})
		lines <- collected
	}()

	var got []string
	Eventually(lines, "5s").Should(Receive(&got))
	Expect(got).To(HaveLen(2))
	Expect(got[1]).To(Equal("__NODE__:after"))
}

// TestDrainPipeShortLineUnchanged confirms ordinary small lines pass through
// untouched.
func TestDrainPipeShortLineUnchanged(t *testing.T) {
	RegisterTestingT(t)

	var lines []string
	drainPipe(strings.NewReader("hello\nworld\n"), maxStoredLineBytes, func(line string) {
		lines = append(lines, line)
	})

	Expect(lines).To(Equal([]string{"hello", "world"}))
}
