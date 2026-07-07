package sandbox

import "strings"

// buildCommandLine renders cmd as a single shell-quoted line, followed by a
// line that echoes marker and the command's exit code so the reader side can
// find the boundary of this command's output in the shell's endless stream.
func buildCommandLine(cmd []string, marker string, cdTo string) string {
	quoted := make([]string, len(cmd))
	for i, arg := range cmd {
		quoted[i] = shQuote(arg)
	}
	line := strings.Join(quoted, " ")
	if cdTo != "" {
		line = "cd " + shQuote(cdTo) + " && " + line
	}
	return line + "\necho \"" + marker + " $?\"\n"
}

// shQuote wraps s in single quotes so a shell treats it as one literal word,
// immune to word-splitting and globbing — the same guarantee argv exec gave
// callers before commands were routed through a shell.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
