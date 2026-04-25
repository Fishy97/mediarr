# Local AI Guide

Mediarr can run with an optional Ollama sidecar. AI is never required for scanning, catalog persistence, backup/restore, or deterministic recommendations.

## Launch Modes

```bash
# No AI
docker compose up -d

# With AI sidecar
docker compose --profile ai up -d
```

The AI profile starts:

- `ollama`, the local model runtime
- `mediarr-ai-init`, a one-shot initializer that pulls `MEDIARR_AI_MODEL`

The default model is `qwen3:0.6b`.

## What AI Can Do

AI can attach a short advisory rationale and tags to recommendations that were already created by deterministic rules.

AI cannot:

- create catalog truth automatically
- delete or move media
- override user corrections
- replace deterministic confidence, source, affected paths, or safety fields

## Reliability Rules

Mediarr validates AI responses as strict JSON with `rationale`, `tags`, and `confidence`. Invalid responses are ignored. If Ollama is unavailable or the model is missing, Mediarr continues without AI rationale.
