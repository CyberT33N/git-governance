package packaging

import (
	"strings"
	"testing"
)

// failClosedPrimitivePrefixes are the lane-authored fail-closed check forms
// that stop a governed lane under set -e; every occurrence outside an
// if/while/until/for condition must carry a named reason on the lane's
// diagnostic surface (the failed check, the value class, the remediation
// pointer).
var failClosedPrimitivePrefixes = []string{
	"test ",
	"[[ ",
	"git cat-file -e",
	"git merge-base --is-ancestor",
	"git ls-remote --exit-code",
	"jq --exit-status",
	"jq -e ",
}

// TestLifecyclePayloadsEmitNamedReasonsAtEveryFailClosedCheck proves that every
// fail-closed check of the lifecycle payload family emits its named reason on
// the lane's own diagnostic surface: a fail-closed stop without its named
// reason is a defect of the lane, never a hardening property.
func TestLifecyclePayloadsEmitNamedReasonsAtEveryFailClosedCheck(t *testing.T) {
	t.Parallel()

	for _, name := range lifecyclePayloads {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			workflow := readWorkflow(t, name)
			blocks := runBlocks(workflow)
			if len(blocks) == 0 {
				t.Fatalf("%s carries no run blocks", name)
			}
			for _, block := range blocks {
				for _, finding := range silentFailClosedChecks(block) {
					t.Errorf("%s: %s", name, finding)
				}
			}
		})
	}
}

// TestNamedReasonGuardRejectsTheSilentDefaultBranchRefCheck pins the proven
// defect form: a bare default-branch ref check stopped the release-request
// lane under set -e without any named reason.
func TestNamedReasonGuardRejectsTheSilentDefaultBranchRefCheck(t *testing.T) {
	t.Parallel()

	findings := silentFailClosedChecks(`test "$GITHUB_REF" = "refs/heads/main"`)
	if len(findings) != 1 {
		t.Fatalf("silentFailClosedChecks() = %v, want exactly one finding for the proven silent ref check", findings)
	}
}

// TestSilentFailClosedChecksForms proves the analyzer branches against the
// canonical named forms, the proven silent forms, and the shell constructs the
// lifecycle payload family uses.
func TestSilentFailClosedChecksForms(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name  string
		shell string
		want  int
	}{
		{
			name:  "the named or-brace form is accepted",
			shell: `test "$A" = "b" || { echo "a must equal b (value class); fix the binding" >&2; exit 1; }`,
			want:  0,
		},
		{
			name: "the named multi-line or-brace form is accepted",
			shell: `test "$STATUS" = "200" || {
  echo "expected 200 (broker response status); verify the broker" >&2
  exit 1
}`,
			want: 0,
		},
		{
			name:  "a bare test is the proven silent defect",
			shell: `test "$GITHUB_REF" = "refs/heads/main"`,
			want:  1,
		},
		{
			name:  "an or-exit without a named reason is rejected twice",
			shell: `test "$STATUS" = "200" || exit 1`,
			want:  2,
		},
		{
			name:  "an or-brace exit without a named reason is rejected twice",
			shell: `test -n "$TOKEN" || { exit 1; }`,
			want:  2,
		},
		{
			name:  "a bare jq extraction is rejected",
			shell: `request_id="$(jq --exit-status --raw-output '.fields.requestID' <<<"$result")"`,
			want:  1,
		},
		{
			name: "the named jq extraction is accepted",
			shell: `request_id="$(jq --exit-status --raw-output '.fields.requestID' <<<"$result")" || {
  echo "missing fields.requestID (CLI JSON output contract); verify the CLI" >&2
  exit 1
}`,
			want: 0,
		},
		{
			name:  "a bare git existence probe is rejected",
			shell: `git cat-file -e "${SHA}^{commit}"`,
			want:  1,
		},
		{
			name: "the named if-wrapped probe is accepted",
			shell: `if ! git cat-file -e "${SHA}^{commit}"; then
  echo "the source revision is not available (source revision); verify the record" >&2
  exit 1
fi`,
			want: 0,
		},
		{
			name: "an if-wrapped probe without a named reason is rejected",
			shell: `if ! git cat-file -e "${SHA}^{commit}"; then
  exit 1
fi`,
			want: 1,
		},
		{
			name: "an if condition is control flow",
			shell: `if test -n "$existing"; then
  echo "present"
fi`,
			want: 0,
		},
		{
			name: "a silent check inside a loop is rejected",
			shell: `for variable in A B; do
  test -n "${!variable:-}"
done`,
			want: 1,
		},
		{
			name: "the named check inside a loop is accepted",
			shell: `for variable in A B; do
  test -n "${!variable:-}" || { echo "missing variable (environment value); set it" >&2; exit 1; }
done`,
			want: 0,
		},
		{
			name: "the toolchain resolver keeps the outer named reason",
			shell: `directive="$(
  awk '{ sub(/\r$/, "") }
       $1 == "toolchain" { count += 1; directive = $2 }
       END {
         if (count != 1) exit 1
         print directive
       }' go.mod
)" || {
  echo "go.mod must carry exactly one pinned toolchain directive (toolchain directive); fix go.mod" >&2
  exit 1
}`,
			want: 0,
		},
		{
			name:  "a bare conditional is rejected",
			shell: `[[ "$A" == "b" ]]`,
			want:  1,
		},
		{
			name:  "the named conditional is accepted",
			shell: `[[ "$A" == "b" ]] || { echo "a must equal b (value); fix it" >&2; exit 1; }`,
			want:  0,
		},
		{
			name:  "a bare ancestor probe is rejected",
			shell: `git merge-base --is-ancestor "$A" "origin/main"`,
			want:  1,
		},
		{
			name:  "a bare ls-remote exit-code probe is rejected",
			shell: `git ls-remote --exit-code --tags origin "refs/tags/v1"`,
			want:  1,
		},
		{
			name: "the named ls-remote probe in a condition is accepted",
			shell: `if git ls-remote --exit-code --tags origin "refs/tags/v1" >/dev/null 2>&1; then
  echo "tag exists (tag ref); pick a new version" >&2
  exit 1
fi`,
			want: 0,
		},
		{
			name:  "an informational echo is not a check",
			shell: `echo "resolved the pinned toolchain directive: $directive"`,
			want:  0,
		},
		{
			name: "a zero exit is not fail-closed",
			shell: `if test -n "$existing"; then
  test "$existing" = "$SOURCE" || {
    echo "unauthorized revision (source revision); re-authorize" >&2
    exit 1
  }
  exit 0
fi`,
			want: 0,
		},
		{
			name: "the github expression and parentheses inside the named reason are accepted",
			shell: `test "$(go env GOVERSION)" = "go${{ steps.toolchain.outputs.version }}" || {
  echo "requires the pinned toolchain (toolchain version); verify setup-go" >&2
  exit 1
}`,
			want: 0,
		},
		{
			name: "a stdout echo does not satisfy the diagnostic form",
			shell: `if ! git cat-file -e "${SHA}^{commit}"; then
  echo "missing"
  exit 1
fi`,
			want: 1,
		},
		{
			name: "the multi-line conditional chain is accepted",
			shell: `[[ "$TARGET" == "develop" ||
   "$TARGET" =~ ^release/[0-9]+$ ||
   "$TARGET" =~ ^support/[0-9]+$ ]] || {
  echo "target must be a shared line (target line); fix the input" >&2
  exit 1
}`,
			want: 0,
		},
		{
			name: "the subshell content is not a check",
			shell: `(
  cd "$(dirname "${checksum}")"
  sha256sum --check "$(basename "${checksum}")"
)`,
			want: 0,
		},
		{
			name:  "a masked output echo without a check is accepted",
			shell: `echo "::add-mask::$token"`,
			want:  0,
		},
		{
			name:  "a check followed by an unrelated diagnostic without the or-boundary is rejected",
			shell: `test "$A" = "b"; echo "unrelated note" >&2`,
			want:  1,
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := silentFailClosedChecks(testCase.shell); len(got) != testCase.want {
				t.Fatalf("silentFailClosedChecks() = %v, want %d findings", got, testCase.want)
			}
		})
	}
}

// runBlocks extracts the shell content of every `run: |` literal block of a
// workflow document.
func runBlocks(workflow string) []string {
	var blocks []string
	lines := strings.Split(workflow, "\n")
	for index := 0; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if !strings.HasPrefix(trimmed, "run: |") {
			continue
		}
		indent := len(lines[index]) - len(strings.TrimLeft(lines[index], " "))
		var block []string
		for index+1 < len(lines) {
			next := lines[index+1]
			if strings.TrimSpace(next) != "" &&
				len(next)-len(strings.TrimLeft(next, " ")) <= indent {
				break
			}
			block = append(block, next)
			index++
		}
		blocks = append(blocks, strings.Join(block, "\n"))
	}
	return blocks
}

// logicalLines joins backslash line continuations of a shell block.
func logicalLines(block string) []string {
	var lines []string
	var current strings.Builder
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimRight(line, " \t")
		if strings.HasSuffix(trimmed, "\\") {
			current.WriteString(strings.TrimSuffix(trimmed, "\\"))
			continue
		}
		current.WriteString(line)
		lines = append(lines, current.String())
		current.Reset()
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}

// shellScanState tracks the quoting and nesting state while a shell block is
// scanned for statement boundaries.
type shellScanState struct {
	inSingle      bool
	inDouble      bool
	parenDepth    int
	braceDepth    int
	paramDepth    int
	compoundDepth int
	atCommand     bool
}

// complete reports whether the accumulated content forms a complete statement
// at a line boundary.
func (state *shellScanState) complete() bool {
	return !state.inSingle && !state.inDouble && state.parenDepth == 0 &&
		state.braceDepth == 0 && state.paramDepth == 0 && state.compoundDepth == 0
}

// shellStatements splits a shell block into logical statements: quote spans,
// command substitutions, parentheses, parameter expansions, brace groups, and
// the compound commands if/for/while/until/case keep a statement open until
// they close.
func shellStatements(block string) []string {
	var statements []string
	var current strings.Builder
	state := shellScanState{}

	for _, line := range logicalLines(block) {
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
		state.atCommand = true
		scanShellLine(line, &state)
		if state.complete() && !hasTrailingContinuation(line) {
			if strings.TrimSpace(current.String()) != "" {
				statements = append(statements, current.String())
			}
			current.Reset()
		}
	}
	if strings.TrimSpace(current.String()) != "" {
		statements = append(statements, current.String())
	}
	return statements
}

// scanShellLine tracks the quoting, nesting, and compound-command state across
// one logical line. Inside double quotes only the escape and the closing quote
// are tracked; the family keeps every quoted span balanced within its line, so
// the continuation outcome stays exact for the payload forms.
func scanShellLine(line string, state *shellScanState) {
	for index := 0; index < len(line); index++ {
		ch := line[index]
		if state.inSingle {
			if ch == '\'' {
				state.inSingle = false
			}
			continue
		}
		if state.inDouble {
			if ch == '\\' {
				index++
				continue
			}
			if ch == '"' {
				state.inDouble = false
			}
			continue
		}
		switch ch {
		case '\'':
			state.inSingle = true
		case '"':
			state.inDouble = true
		case '\\':
			index++
		case '$':
			if index+1 < len(line) && line[index+1] == '(' {
				state.parenDepth++
				state.atCommand = true
				index++
			} else if index+1 < len(line) && line[index+1] == '{' {
				state.paramDepth++
				index++
			}
		case '(':
			state.parenDepth++
			state.atCommand = true
		case ')':
			if state.parenDepth > 0 {
				state.parenDepth--
			}
		case '{':
			if state.paramDepth > 0 {
				state.paramDepth++
			} else {
				state.braceDepth++
			}
			state.atCommand = true
		case '}':
			if state.paramDepth > 0 {
				state.paramDepth--
			} else if state.braceDepth > 0 {
				state.braceDepth--
			}
		case ';', '|':
			state.atCommand = true
		case '&':
			// a redirection operator (>& or N>& or &>) is not a command
			// separator
			if index > 0 && (line[index-1] == '>' || isShellDigit(line[index-1])) {
				continue
			}
			if index+1 < len(line) && line[index+1] == '>' {
				continue
			}
			state.atCommand = true
		default:
			if state.atCommand {
				if isShellWordChar(ch) {
					end := index
					for end < len(line) && isShellWordChar(line[end]) {
						end++
					}
					state.trackKeyword(line[index:end], line, end)
					index = end - 1
				} else if !isShellSpace(ch) {
					state.atCommand = false
				}
			}
		}
	}
}

// trackKeyword maintains the compound-command depth for the shell keywords
// if/for/while/until/case and their terminators fi/done/esac; a word counts
// only at a command position with a word boundary after it.
func (state *shellScanState) trackKeyword(word string, line string, end int) {
	if end < len(line) && !isShellBoundary(line[end]) {
		return
	}
	switch word {
	case "if", "for", "while", "until", "case":
		state.compoundDepth++
	case "fi", "done", "esac":
		if state.compoundDepth > 0 {
			state.compoundDepth--
		}
	}
}

// hasTrailingContinuation reports whether a logical line ends with a shell
// continuation operator.
func hasTrailingContinuation(line string) bool {
	trimmed := strings.TrimRight(line, " \t")
	for _, marker := range []string{"||", "&&", "|", " then", " do", " else"} {
		if strings.HasSuffix(trimmed, marker) {
			return true
		}
	}
	return false
}

// isShellWordChar reports whether a byte continues a shell keyword.
func isShellWordChar(ch byte) bool {
	return ch >= 'a' && ch <= 'z'
}

// isShellSpace reports whether a byte is inline shell whitespace.
func isShellSpace(ch byte) bool {
	return ch == ' ' || ch == '\t'
}

// isShellDigit reports whether a byte is a decimal digit (the redirection
// descriptor class).
func isShellDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// isShellBoundary reports whether a byte ends a shell keyword.
func isShellBoundary(ch byte) bool {
	switch ch {
	case ' ', '\t', ';', '|', '&', '{', '}', '(', ')':
		return true
	}
	return false
}

// shellCommand is one command inside a shell statement, with its byte offsets
// in the statement.
type shellCommand struct {
	start int
	end   int
	text  string
}

// splitCommands splits one shell statement into its commands at the command
// separators, quote- and expansion-aware; commands inside command
// substitutions are reported as their own commands with their absolute
// offsets.
func splitCommands(statement string) []shellCommand {
	return splitCommandsAt(statement, 0)
}

func splitCommandsAt(statement string, base int) []shellCommand {
	var commands []shellCommand
	inSingle, inDouble := false, false
	start := 0
	flush := func(end int) {
		raw := statement[start:end]
		text := strings.TrimSpace(raw)
		if text == "" {
			return
		}
		lead := len(raw) - len(strings.TrimLeft(raw, " \t\n"))
		commands = append(commands, shellCommand{
			start: base + start + lead,
			end:   base + start + lead + len(text),
			text:  text,
		})
	}

	for index := 0; index < len(statement); index++ {
		ch := statement[index]
		if inSingle {
			if ch == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if ch == '\\' {
				index++
				continue
			}
			if ch == '"' {
				inDouble = false
				continue
			}
			if ch == '$' && index+1 < len(statement) && statement[index+1] == '(' {
				close := matchParen(statement, index+1)
				commands = append(commands, splitCommandsAt(statement[index+2:close], base+index+2)...)
				index = close
			}
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '\\':
			index++
		case '$':
			if index+1 < len(statement) && statement[index+1] == '(' {
				close := matchParen(statement, index+1)
				commands = append(commands, splitCommandsAt(statement[index+2:close], base+index+2)...)
				index = close
			}
		case '(':
			close := matchParen(statement, index)
			commands = append(commands, splitCommandsAt(statement[index+1:close], base+index+1)...)
			index = close
		case ';', '\n', '{', '}':
			flush(index)
			start = index + 1
		case '|':
			flush(index)
			if index+1 < len(statement) && statement[index+1] == '|' {
				index++
			}
			start = index + 1
		case '&':
			// a redirection operator (>& or N>& or &>) is not a command
			// separator
			if index > 0 && (statement[index-1] == '>' || isShellDigit(statement[index-1])) {
				continue
			}
			if index+1 < len(statement) && statement[index+1] == '>' {
				continue
			}
			flush(index)
			if index+1 < len(statement) && statement[index+1] == '&' {
				index++
			}
			start = index + 1
		}
	}
	flush(len(statement))
	return commands
}

// matchParen returns the index of the closing parenthesis matching the opening
// parenthesis at open, quote-aware; unbalanced input returns the end of the
// text.
func matchParen(text string, open int) int {
	depth := 0
	inSingle, inDouble := false, false
	for index := open; index < len(text); index++ {
		ch := text[index]
		if inSingle {
			if ch == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if ch == '\\' {
				index++
				continue
			}
			if ch == '"' {
				inDouble = false
			}
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '\\':
			index++
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return len(text)
}

// isCompoundHead reports whether a command opens or closes a compound command
// or forms its condition; the condition of an if/while/until/for is control
// flow, never a fail-closed check of the lane.
func isCompoundHead(text string) bool {
	trimmed := strings.TrimLeft(text, " \t\n")
	for _, prefix := range []string{"if ", "while ", "until ", "for ", "case ", "elif "} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	for _, word := range []string{"then", "do", "else", "fi", "done", "esac"} {
		if trimmed == word || strings.HasPrefix(trimmed, word+" ") {
			return true
		}
	}
	return false
}

// hasAnyPrefix reports whether the text starts with any of the given
// prefixes.
func hasAnyPrefix(text string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

// isFailClosedPrimitive reports whether a command is a lane-authored
// fail-closed check form.
func isFailClosedPrimitive(text string) bool {
	trimmed := strings.TrimLeft(text, " \t\n")
	for strings.HasPrefix(trimmed, "! ") {
		trimmed = strings.TrimLeft(strings.TrimPrefix(trimmed, "! "), " \t\n")
	}
	for _, prefix := range failClosedPrimitivePrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// isStderrDiagnostic reports whether a command writes a diagnostic message to
// stderr; the redirection must be unquoted.
func isStderrDiagnostic(text string) bool {
	trimmed := strings.TrimLeft(text, " \t\n")
	if !strings.HasPrefix(trimmed, "echo ") {
		return false
	}
	inSingle, inDouble := false, false
	for index := 0; index+2 < len(trimmed); index++ {
		ch := trimmed[index]
		if inSingle {
			if ch == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if ch == '\\' {
				index++
				continue
			}
			if ch == '"' {
				inDouble = false
			}
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '\\':
			index++
		case '>':
			if trimmed[index+1] == '&' && trimmed[index+2] == '2' {
				return true
			}
		}
	}
	return false
}

// isFailExit reports whether a command is an explicit non-zero exit.
func isFailExit(text string) bool {
	trimmed := strings.TrimLeft(text, " \t\n")
	if trimmed != "exit" && !strings.HasPrefix(trimmed, "exit ") {
		return false
	}
	rest := strings.TrimLeft(strings.TrimPrefix(trimmed, "exit"), " \t\n")
	return rest != "" && rest[0] >= '1' && rest[0] <= '9'
}

// silentFailClosedChecks analyses one shell block and reports every
// fail-closed check without a named reason and every non-zero exit without a
// preceding named reason on the lane's diagnostic surface.
func silentFailClosedChecks(block string) []string {
	var findings []string
	for _, statement := range shellStatements(block) {
		commands := splitCommands(statement)

		primitiveReported := false
		conditionDepth := 0
		for _, command := range commands {
			trimmed := strings.TrimLeft(command.text, " \t\n")
			// strip leading then/do/else markers so their trailing content is
			// still analyzed
			for {
				if trimmed == "then" || trimmed == "do" || trimmed == "else" {
					if conditionDepth > 0 && trimmed != "else" {
						conditionDepth--
					}
					trimmed = ""
					break
				}
				if hasAnyPrefix(trimmed, "then ", "do ", "else ") {
					word, _, found := strings.Cut(trimmed, " ")
					if conditionDepth > 0 && found && word != "else" {
						conditionDepth--
					}
					trimmed = strings.TrimLeft(trimmed[len(word):], " \t\n")
					continue
				}
				break
			}
			if trimmed == "" {
				continue
			}
			if hasAnyPrefix(trimmed, "if ", "while ", "until ", "elif ", "for ") {
				conditionDepth++
				continue
			}
			if conditionDepth > 0 || isCompoundHead(trimmed) || !isFailClosedPrimitive(trimmed) {
				continue
			}
			covered := false
			for _, diagnostic := range commands {
				if diagnostic.start > command.end && isStderrDiagnostic(diagnostic.text) &&
					strings.Contains(statement[command.end:diagnostic.start], "||") {
					covered = true
					break
				}
			}
			if !covered && !primitiveReported {
				findings = append(findings, "fail-closed check without a named reason: "+summarizeStatement(statement))
				primitiveReported = true
			}
		}

		for _, command := range commands {
			if !isFailExit(command.text) {
				continue
			}
			preceded := false
			for _, candidate := range commands {
				if candidate.start < command.start && isStderrDiagnostic(candidate.text) {
					preceded = true
					break
				}
			}
			if !preceded {
				findings = append(findings, "fail-closed exit without a preceding named reason: "+summarizeStatement(statement))
			}
		}
	}
	return findings
}

// summarizeStatement compacts a shell statement for a finding message.
func summarizeStatement(statement string) string {
	compact := strings.Join(strings.Fields(statement), " ")
	if len(compact) > 160 {
		compact = compact[:160] + "..."
	}
	return compact
}
