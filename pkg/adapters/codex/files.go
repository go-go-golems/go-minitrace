package codex

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/go-minitrace/pkg/minitrace"
)

func codexFileTarget(path, operation, evidence, cwd, reference string) minitrace.FileTarget {
	target := minitrace.FileTarget{Path: path, NativePath: path, OperationType: operation, EvidenceKind: evidence, Status: "attempted", SourceReference: reference}
	if strings.HasPrefix(cwd, "file:") {
		parsed, err := url.Parse(cwd)
		if err == nil && (parsed.Host == "" || parsed.Host == "localhost") {
			cwd = parsed.Path
		} else {
			cwd = ""
		}
	}
	if filepath.IsAbs(cwd) {
		target.CWD = filepath.Clean(cwd)
	}
	if filepath.IsAbs(path) {
		target.Path = filepath.Clean(path)
		target.Resolved = true
	} else if target.CWD != "" {
		target.Path = filepath.Join(target.CWD, path)
		target.Resolved = true
	}
	return target
}

func addCodexFileDiagnostic(call *minitrace.ToolCall, code string) {
	metadata := mapValue(call.FrameworkMetadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	codes, _ := metadata["fidelity_diagnostics"].([]string)
	for _, existing := range codes {
		if existing == code {
			return
		}
	}
	if len(codes) < 32 {
		metadata["fidelity_diagnostics"] = append(codes, code)
	}
	call.FrameworkMetadata = metadata
}

func finalizeCodexFileEvidence(calls []minitrace.ToolCall, sourcePath string) {
	for index := range calls {
		call := &calls[index]
		call.RecordKind = call.EffectiveRecordKind()
		if call.RecordKind == minitrace.RecordKindFileChange {
			for i := range call.Input.FileTargets {
				if strings.HasPrefix(call.Input.FileTargets[i].SourceReference, "line:") {
					call.Input.FileTargets[i].SourceReference = sourcePath + "#L" + strings.TrimPrefix(call.Input.FileTargets[i].SourceReference, "line:")
				}
			}
			continue
		}
		call.Input.FileTargets = []minitrace.FileTarget{}
		// Never retain regex-derived convenience paths for Codex; derive them only
		// from the same structural evidence as the multi-target ledger.
		call.Input.FilePath = nil
		if call.RecordKind == minitrace.RecordKindOrchestration {
			continue
		}
		metadata := mapValue(call.FrameworkMetadata)
		args := mapValue(call.Input.Arguments)
		cwd := firstNonEmpty(stringValue(args["cwd"]), stringValue(args["workdir"]), stringValue(metadata["native_cwd"]))
		line, _ := codexInteger(metadata["source_line"])
		reference := ""
		if line > 0 {
			reference = fmt.Sprintf("%s#L%d", sourcePath, line)
		}
		switch call.ToolName {
		case "apply_patch":
			targets, diagnostic := codexPatchTargets(stringValue(args["input"]))
			if diagnostic != "" {
				addCodexFileDiagnostic(call, diagnostic)
			}
			for _, target := range targets {
				call.Input.FileTargets = append(call.Input.FileTargets, codexFileTarget(target.path, target.operation, "patch_header", cwd, reference))
			}
		case "exec_command":
			if metadata["command_rendering"] == "quoted_argv_display" || call.Input.Command == nil {
				continue
			}
			targets, diagnostic := literalShellTargets(*call.Input.Command)
			if diagnostic != "" {
				addCodexFileDiagnostic(call, diagnostic)
			}
			for _, target := range targets {
				call.Input.FileTargets = append(call.Input.FileTargets, codexFileTarget(target.path, target.operation, "shell_redirect", cwd, reference))
			}
		case "read_file", "view_image", "write_file", "edit_file", "apply_diff":
			path := firstNonEmpty(stringValue(args["file_path"]), stringValue(args["path"]))
			if path != "" {
				call.Input.FileTargets = append(call.Input.FileTargets, codexFileTarget(path, call.OperationType, "tool_path_argument", cwd, reference))
			}
		}
		if len(call.Input.FileTargets) > 0 {
			call.Input.FilePath = ptr(call.Input.FileTargets[0].Path)
		}
	}
}

func codexPatchTargets(patch string) ([]shellTarget, string) {
	if len(patch) > 1024*1024 {
		return nil, "patch_analysis_size_limit"
	}
	lines := strings.Split(strings.TrimSpace(patch), "\n")
	if len(lines) < 2 || lines[0] != "*** Begin Patch" || lines[len(lines)-1] != "*** End Patch" {
		return nil, "unsupported_patch_envelope"
	}
	var targets []shellTarget
	for _, line := range lines[1 : len(lines)-1] {
		operation, path := "", ""
		switch {
		case strings.HasPrefix(line, "*** Add File: "):
			operation = "NEW"
			path = strings.TrimPrefix(line, "*** Add File: ")
		case strings.HasPrefix(line, "*** Update File: "):
			operation = "MODIFY"
			path = strings.TrimPrefix(line, "*** Update File: ")
		case strings.HasPrefix(line, "*** Delete File: "):
			operation = "DELETE"
			path = strings.TrimPrefix(line, "*** Delete File: ")
		case strings.HasPrefix(line, "*** Move to: "):
			if len(targets) == 0 || targets[len(targets)-1].operation != "MODIFY" {
				return nil, "invalid_patch_move"
			}
			targets[len(targets)-1].operation = "DELETE"
			operation = "NEW"
			path = strings.TrimPrefix(line, "*** Move to: ")
		}
		if operation != "" {
			if path == "" {
				return nil, "invalid_patch_path"
			}
			targets = append(targets, shellTarget{path, operation})
		}
	}
	return targets, ""
}
