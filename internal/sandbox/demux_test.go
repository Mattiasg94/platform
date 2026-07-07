package sandbox

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// frame builds one Docker multiplexed-stream frame: an 8-byte header
// (stream type + big-endian payload size) followed by the payload.
func frame(streamType byte, payload string) []byte {
	header := make([]byte, 8)
	header[0] = streamType
	binary.BigEndian.PutUint32(header[4:8], uint32(len(payload)))
	return append(header, payload...)
}

func TestDemuxUntilMarker_ReturnsStdoutAndExitCode(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(frame(1, "hello\n"))
	buf.Write(frame(1, "MARK123 0\n"))

	stdout, stderr, exitCode, err := demuxUntilMarker(&buf, "MARK123")
	if err != nil {
		t.Fatalf("demuxUntilMarker: %v", err)
	}
	if stdout != "hello\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "hello\n")
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
}

func TestDemuxUntilMarker_KeepsStderrSeparateFromStdout(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(frame(1, "out line\n"))
	buf.Write(frame(2, "err line\n"))
	buf.Write(frame(1, "MARK123 1\n"))

	stdout, stderr, exitCode, err := demuxUntilMarker(&buf, "MARK123")
	if err != nil {
		t.Fatalf("demuxUntilMarker: %v", err)
	}
	if stdout != "out line\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "out line\n")
	}
	if stderr != "err line\n" {
		t.Fatalf("stderr = %q, want %q", stderr, "err line\n")
	}
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
}

func TestDemuxUntilMarker_HandlesMarkerLineSplitAcrossFrames(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(frame(1, "hello\n"))
	buf.Write(frame(1, "MARK123 "))
	buf.Write(frame(1, "0\n"))

	stdout, _, exitCode, err := demuxUntilMarker(&buf, "MARK123")
	if err != nil {
		t.Fatalf("demuxUntilMarker: %v", err)
	}
	if stdout != "hello\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "hello\n")
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
}

func TestDemuxUntilMarker_IgnoresMarkerLookalikeLines(t *testing.T) {
	var buf bytes.Buffer
	// Command's own output mentions the marker text, but not as a
	// standalone "<marker> <exitcode>" line — must not end the read early.
	buf.Write(frame(1, "saw MARK123 in the logs\n"))
	buf.Write(frame(1, "MARK123 extra stuff 0\n"))
	buf.Write(frame(1, "MARK123 0\n"))

	stdout, _, exitCode, err := demuxUntilMarker(&buf, "MARK123")
	if err != nil {
		t.Fatalf("demuxUntilMarker: %v", err)
	}
	want := "saw MARK123 in the logs\nMARK123 extra stuff 0\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
}
