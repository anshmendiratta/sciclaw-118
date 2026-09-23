# Independent review: safe provider errors

Review `origin/main...codex/safe-provider-errors` independently. Do not trust
the implementation summary. Submit findings by severity with file and line
references; approve only when no material findings remain.

Verify:

1. Raw HTTP/provider bodies remain available only in server-side logs and are
   correlated with the user-visible `ERR-...` reference.
2. User-visible Discord replies, async job cards, and `/api/chat` responses do
   not expose provider bodies, credentials, URLs, prompts, or stack traces.
3. The error classifier maps authentication, quota, rate-limit, model,
   context, timeout, and 5xx failures to actionable deterministic messages.
4. The JSON web-chat path remains valid when the agent exits nonzero, and
   technical details are separate from the primary response.
5. Discord technical details are rendered as spoiler text and contain only the
   allowlisted sanitized fields.
6. Existing non-error outbound messages and provider success responses are
   unaffected.
7. Tests fail for the previously raw provider-error behavior and cover the
   important non-leakage path.

Verification run by the implementer: `go test ./...` and `git diff --check`.
