# Soak Eval

Soak eval runs a long DND session against the real backend message API and records per-round behavior plus aggregate metrics. The sample config at `configs/eval/soak_the_city_50.json` uses `${EVAL_*}` placeholders only; set real values in the shell or CI secret store.

## Example Environment

```sh
export EVAL_BASE_URL="http://localhost:8080"
export EVAL_SESSION_ID="soak-the-city-50"
export EVAL_USER_TOKEN="<token>"
export EVAL_PLAYER_PROVIDER="<provider>"
export EVAL_PLAYER_MODEL="<model>"
export EVAL_PLAYER_API_KEY="<api-key>"
export EVAL_PLAYER_BASE_URL="<model-base-url>"
export EVAL_JUDGE_PROVIDER="<provider>"
export EVAL_JUDGE_MODEL="<model>"
export EVAL_JUDGE_API_KEY="<api-key>"
export EVAL_JUDGE_BASE_URL="<model-base-url>"
```

Do not write real API keys into the JSON config.

## Run

Start the backend normally, create or choose a session, then run:

```sh
GOCACHE=/tmp/go-build go run ./cmd/soak_eval \
  --config configs/eval/soak_the_city_50.json \
  --output reports/eval/soak_the_city_50.json
```

The evaluator writes both:

```text
reports/eval/soak_the_city_50.json
reports/eval/soak_the_city_50.md
```

The backend currently authenticates session APIs through the `dnd_auth_session` cookie. `EVAL_USER_TOKEN` should therefore be the auth session token value. The HTTP client also sends `Authorization: Bearer <token>` for compatibility with future auth middleware.

## Reports

`WriteReport(path, report)` writes an indented JSON report and creates the parent directory when needed. The JSON report keeps the full `SoakReport`, including all `RoundRecord` entries.

`WriteMarkdownReport(path, report)` writes a concise Markdown summary with:

- session ID, round count, success count, success rate, and average latency
- sorted failure reason counts
- a compact per-round table with success, score, latency, and failure reasons

Typical output paths:

```text
reports/eval/soak_the_city_50.json
reports/eval/soak_the_city_50.md
```

## Local Verification

Run the focused tests with a writable Go build cache:

```sh
GOCACHE=/tmp/go-build go test ./internal/eval/soak ./cmd/soak_eval
```
