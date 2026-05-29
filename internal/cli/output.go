// Package cli implements the logify CLI. Output is plain text by default —
// dense, line-oriented, the shape LLMs read fastest. `--json` (global flag)
// switches to structured JSON / NDJSON. Exit codes carry meaning regardless.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// EmitJSON is set by the global --json flag before dispatch. When false (default)
// every command renders text; when true, JSON.
var EmitJSON = false

// Exit codes — must match SKILL.md.
const (
	ExitOK           = 0
	ExitGeneric      = 1
	ExitBadInput     = 2
	ExitNotFound     = 3
	ExitAmbiguous    = 4
	ExitAuth         = 5
	ExitNetwork      = 6
	ExitNotAllowed   = 7
	ExitNotReachable = 8
)

// CLIError is the JSON shape emitted on non-zero exit.
type CLIError struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Hint       string         `json:"hint,omitempty"`
	Candidates []ErrCandidate `json:"candidates,omitempty"`
}

type ErrCandidate struct {
	Path string `json:"path"`
	UUID string `json:"uuid"`
}

// emitJSON writes v to stdout as compact JSON + a trailing newline.
func emitJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// emitErr writes an error and returns the matching exit code. In JSON mode
// it emits `{"error": {...}}`; in text mode a dense multi-line block.
func emitErr(code int, body CLIError) int {
	if EmitJSON {
		_ = emitJSON(map[string]CLIError{"error": body})
	} else {
		fmt.Fprintln(os.Stdout, formatErrText(body))
	}
	return code
}

func formatErrText(e CLIError) string {
	var b strings.Builder
	b.WriteString("error: ")
	if e.Code != "" {
		b.WriteString(e.Code)
		b.WriteString(" — ")
	}
	b.WriteString(e.Message)
	if e.Hint != "" {
		b.WriteString("\nhint:  ")
		b.WriteString(e.Hint)
	}
	if len(e.Candidates) > 0 {
		b.WriteString("\ncandidates:")
		// align candidates' uuid column
		maxPath := 0
		for _, c := range e.Candidates {
			if len(c.Path) > maxPath {
				maxPath = len(c.Path)
			}
		}
		for _, c := range e.Candidates {
			pad := strings.Repeat(" ", maxPath-len(c.Path))
			b.WriteString("\n  ")
			b.WriteString(c.Path)
			if c.UUID != "" {
				b.WriteString(pad)
				b.WriteString("  ")
				b.WriteString(c.UUID)
			}
		}
	}
	return b.String()
}

// emitNDJSON writes one JSON object per line to the given writer.
func emitNDJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// errBadInput is a convenience for flag/usage errors.
func errBadInput(format string, args ...any) int {
	return emitErr(ExitBadInput, CLIError{Code: "BAD_INPUT", Message: fmt.Sprintf(format, args...)})
}

// codeFor maps a CLIError code string back to its exit code.
func codeFor(c string) int {
	switch c {
	case "NOT_FOUND":
		return ExitNotFound
	case "AMBIGUOUS":
		return ExitAmbiguous
	case "AUTH":
		return ExitAuth
	case "NETWORK":
		return ExitNetwork
	case "NOT_ALLOWED":
		return ExitNotAllowed
	case "NOT_REACHABLE":
		return ExitNotReachable
	case "BAD_INPUT":
		return ExitBadInput
	}
	return ExitGeneric
}
