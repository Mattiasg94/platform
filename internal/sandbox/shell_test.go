package sandbox

import "testing"

func TestBuildCommandLine_JoinsArgvWithMarkerEcho(t *testing.T) {
	got := buildCommandLine([]string{"echo", "hi"}, "MARK123", "")

	want := "'echo' 'hi'\necho \"MARK123 $?\"\n"
	if got != want {
		t.Fatalf("buildCommandLine() = %q, want %q", got, want)
	}
}

func TestBuildCommandLine_QuotesArgsSoSpacesStayLiteral(t *testing.T) {
	got := buildCommandLine([]string{"echo", "hello world"}, "MARK123", "")

	want := "'echo' 'hello world'\necho \"MARK123 $?\"\n"
	if got != want {
		t.Fatalf("buildCommandLine() = %q, want %q", got, want)
	}
}

func TestBuildCommandLine_EscapesEmbeddedSingleQuote(t *testing.T) {
	got := buildCommandLine([]string{"echo", "it's"}, "MARK123", "")

	want := "'echo' 'it'\\''s'\necho \"MARK123 $?\"\n"
	if got != want {
		t.Fatalf("buildCommandLine() = %q, want %q", got, want)
	}
}

func TestBuildCommandLine_PrefixesCdWhenCdToGiven(t *testing.T) {
	got := buildCommandLine([]string{"pwd"}, "MARK123", "/workspace/sub")

	want := "cd '/workspace/sub' && 'pwd'\necho \"MARK123 $?\"\n"
	if got != want {
		t.Fatalf("buildCommandLine() = %q, want %q", got, want)
	}
}
