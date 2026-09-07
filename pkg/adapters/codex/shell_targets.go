package codex

import "strings"

// This deliberately narrow grammar accepts straight-line shell words,
// quoting/escaping, statement boundaries and literal <, > and >> redirects.
// It never evaluates a command. Unsupported control flow, expansion, heredocs,
// pipelines and cwd-changing constructs invalidate the whole analysis.
type shellTarget struct{ path, operation string }
type shellWord struct {
	text     string
	operator bool
}

func literalShellTargets(script string) ([]shellTarget, string) {
	if len(script) > 256*1024 {
		return nil, "shell_analysis_size_limit"
	}
	words, diagnostic := shellWords(script)
	if diagnostic != "" {
		return nil, diagnostic
	}
	var targets []shellTarget
	commandStart := true
	for i := 0; i < len(words); i++ {
		word := words[i]
		if word.operator {
			switch word.text {
			case ";", "\n":
				commandStart = true
			case ">", ">>", "<":
				if i+1 == len(words) || words[i+1].operator || words[i+1].text == "" {
					return nil, "invalid_shell_redirect"
				}
				i++
				operation := "MODIFY"
				if word.text == "<" {
					operation = "READ"
				}
				targets = append(targets, shellTarget{words[i].text, operation})
			}
			continue
		}
		if commandStart {
			switch word.text {
			case "if", "then", "else", "elif", "fi", "for", "while", "until", "case", "select", "function", "do", "done", "!", "time", "cd", "pushd", "popd", "eval", "source", ".":
				return nil, "unsupported_shell_control_or_cwd"
			}
			// Assignments can modify expansion/dispatch semantics; do not infer.
			if strings.Contains(word.text, "=") {
				return nil, "unsupported_shell_assignment"
			}
			// A numeric descriptor immediately before a redirect is not a command.
			if i+1 < len(words) && words[i+1].operator && (words[i+1].text == ">" || words[i+1].text == ">>" || words[i+1].text == "<") && allDigits(word.text) {
				continue
			}
			commandStart = false
		}
	}
	return targets, ""
}

func allDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func shellWords(script string) ([]shellWord, string) {
	var words []shellWord
	for i := 0; i < len(script); {
		c := script[i]
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '#' {
			for i < len(script) && script[i] != '\n' {
				i++
			}
			continue
		}
		if c == ';' || c == '\n' {
			words = append(words, shellWord{text: string(c), operator: true})
			i++
			continue
		}
		if c == '>' || c == '<' {
			op := string(c)
			i++
			if i < len(script) && script[i] == c {
				op += string(c)
				i++
			}
			if op == "<<" {
				return nil, "unsupported_shell_heredoc"
			}
			if i < len(script) && (script[i] == '&' || script[i] == '|' || script[i] == '(' || script[i] == '>') {
				return nil, "unsupported_shell_redirect"
			}
			words = append(words, shellWord{text: op, operator: true})
			continue
		}
		var word strings.Builder
		for i < len(script) {
			c = script[i]
			if strings.ContainsRune(" \t\r\n;><", rune(c)) {
				break
			}
			if strings.ContainsRune("&|(){}$`*?[]~", rune(c)) {
				return nil, "unsupported_shell_expansion_or_control"
			}
			if c == '\\' {
				i++
				if i == len(script) {
					return nil, "invalid_shell_escape"
				}
				if script[i] != '\n' {
					word.WriteByte(script[i])
				}
				i++
				continue
			}
			if c == '\'' || c == '"' {
				quote := c
				i++
				for i < len(script) && script[i] != quote {
					if quote == '"' && (script[i] == '$' || script[i] == '`') {
						return nil, "unsupported_shell_expansion_or_control"
					}
					if quote == '"' && script[i] == '\\' && i+1 < len(script) {
						next := script[i+1]
						if strings.ContainsRune("$`\"\\\n", rune(next)) {
							i++
							if next == '\n' {
								i++
								continue
							}
						}
					}
					word.WriteByte(script[i])
					i++
				}
				if i == len(script) {
					return nil, "invalid_shell_quote"
				}
				i++
				continue
			}
			word.WriteByte(c)
			i++
		}
		words = append(words, shellWord{text: word.String()})
	}
	return words, ""
}
