# oneshot

Zero-dependency single-file Python script that calls any OpenAI-compatible chat API. Returns raw text — no markdown, no fences. Supports conversation continuity via `-c` flag (state in `~/.oneshot_state.json`).

## Usage

```
oneshot "convert mov to mp4"
oneshot -c "but keep audio"
```

## Setup

Wrapper script on your `PATH`:

```bash
#!/usr/bin/env bash
export ONE_SHOT_BASE_URL=https://example.com/api/
export ONE_SHOT_API_KEY=sk-...
export ONE_SHOT_MODEL=gpt-5.4
# https://github.com/astral-sh/uv
# because Espanso is wierd about paths ¯\(ツ)/¯
export UV=/path/to/uv
export SCRIPT=/path/to/oneshot.py
$UV run $SCRIPT "$@"
```

[Espanso](https://espanso.org/) matchers for system-wide access:

```yaml
# :ai  — one-shot prompt via form
- trigger: ":ai"
  replace: "{{output}}"
  vars:
  - name: form1
    type: form
    params:
      layout: |
        [[prompt]]
  - name: output
    type: shell
    params:
      cmd: "/path/to/oneshot '{{ form1.prompt }}'"

# :cai — continue previous conversation
- trigger: ":cai"
  replace: "{{output}}"
  vars:
  - name: form1
    type: form
    params:
      layout: |
        [[prompt]]
  - name: output
    type: shell
    params:
      cmd: "/path/to/oneshot -c '{{ form1.prompt }}'"
```