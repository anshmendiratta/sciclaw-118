# Safe provider errors closeout

Ready for review.

- Outcome: provider HTTP errors retain raw diagnostics in server-side logs while
  user-facing responses are actionable, sanitized, and correlated by `ERR-...`.
- Architecture: typed HTTP errors preserve status, code, request ID, and raw
  body; the agent classifies them deterministically; Discord uses spoiler text
  for allowlisted technical details; web chat receives a structured error.
- Worktree: `/Users/menzy3/projects/sciclaw-provider-errors`.
- Branch: `codex/safe-provider-errors`.
- Base: `origin/main` at `eb557d5d9f98422ab9c4a54d8013d20dfb006288`.
- Head: `7b5254c` (`docs: record provider error review packet`).
- Verification: `go test ./...` and `git diff --check` passed.
- Excluded: custom web-client toggle rendering, provider retries, and changes
  to non-provider failures.
- Independent review: `docs/ops/review-safe-provider-errors.md`.
- Next gate: independent review and CI, then authorized merge.
