## Bug

`/resume` fails with `SyntaxError: Unterminated string in JSON` when `events.jsonl` contains raw 
Unicode Line Separator (U+2028) or Paragraph Separator (U+2029) characters.

## Root Cause

JavaScript's `JSON.parse()` treats U+2028 and U+2029 as line terminators inside strings, making 
them invalid in unescaped form. Python's JSON parser handles them fine, so the file appears valid 
to external tooling — making the bug hard to diagnose.

When terminal output containing U+2028 (e.g., from commands that include certain Unicode text) is 
captured into the session, the raw codepoint gets written into the JSONL event record without 
escaping.

## Reproduction

2. Exit the session
3. Attempt `/resume <session-id>`
4. Observe: `Error: Session file is corrupted (line N: SyntaxError: Unterminated string in JSON at 
position X)`

## Workaround

Manually escape the characters in the session file:

```python
path = '~/.copilot/session-state/<session-id>/events.jsonl'
with open(path) as f:
    content = f.read()
fixed = content.replace('\u2028', '\\u2028').replace('\u2029', '\\u2029')
with open(path, 'w') as f:
    f.write(fixed)
```

## Expected Behavior

The session writer should escape U+2028 and U+2029 as `\u2028`/`\u2029` when serializing event data 
to `events.jsonl`, so the file remains valid for `JSON.parse()`.

## Environment

- macOS (Darwin)
- Copilot CLI v1.0.4
- Session hit this bug twice in one day from different command outputs
