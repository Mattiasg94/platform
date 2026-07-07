package sandbox

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Docker's multiplexed-stream frame header: 1 byte stream type (1=stdout,
// 2=stderr), 3 bytes unused, 4-byte big-endian payload size.
const (
	streamTypeStdout = 1
	streamTypeStderr = 2
	frameHeaderSize  = 8
)

// demuxUntilMarker reads Docker-multiplexed frames from r, splitting them
// into stdout/stderr, until a line on stdout exactly matches "<marker>
// <exitcode>" — the boundary a single Exec call's output ends at inside the
// persistent shell's otherwise-endless stream. The marker line itself is
// excluded from the returned stdout.
func demuxUntilMarker(r io.Reader, marker string) (stdout, stderr string, exitCode int, err error) {
	var outBuf, errBuf bytes.Buffer
	header := make([]byte, frameHeaderSize)

	for {
		if _, err := io.ReadFull(r, header); err != nil {
			return "", "", 0, fmt.Errorf("sandbox: demux read frame header: %w", err)
		}
		size := binary.BigEndian.Uint32(header[4:8])
		payload := make([]byte, size)
		if _, err := io.ReadFull(r, payload); err != nil {
			return "", "", 0, fmt.Errorf("sandbox: demux read frame payload: %w", err)
		}

		switch header[0] {
		case streamTypeStdout:
			outBuf.Write(payload)
			if cut, code, ok := findMarkerLine(outBuf.String(), marker); ok {
				return outBuf.String()[:cut], errBuf.String(), code, nil
			}
		case streamTypeStderr:
			errBuf.Write(payload)
		}
	}
}

// findMarkerLine looks for a complete line reading "<marker> <exitcode>" in
// buf. It only recognizes a whole, newline-terminated line as the marker —
// output that merely contains the marker text as a substring, or on a line
// with other content, does not match.
func findMarkerLine(buf, marker string) (cut int, exitCode int, ok bool) {
	prefix := marker + " "
	start := 0
	for {
		nl := strings.IndexByte(buf[start:], '\n')
		if nl == -1 {
			return 0, 0, false
		}
		line := buf[start : start+nl]
		if rest, found := strings.CutPrefix(line, prefix); found {
			if code, err := strconv.Atoi(rest); err == nil {
				return start, code, true
			}
		}
		start += nl + 1
	}
}
