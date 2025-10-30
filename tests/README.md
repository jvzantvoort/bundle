# Tests for bundle

This folder contains contract-style integration tests that build the CLI and exercise end-to-end behaviour.

## How tests are organized

- Unit tests live alongside packages (use `go test ./...` to run all unit tests).
- Contract tests are in `tests/contract/*.go` and build the `cmd/bundle` binary and execute it from tests to assert exit codes and JSON output.

## Running contract tests

From the repository root run:

```bash
go test ./tests/contract -v
```

To run a single test use `-run`, for example:

```bash
go test ./tests/contract -run TestCLI_More -v
```

The contract tests build a temporary binary in a test temporary directory, then execute it. Tests assume a Go toolchain (Go 1.20+). If you have multiple Go versions installed, ensure the `go` on your PATH is compatible with the module.

## JSON output and logging

The CLI supports a global `--json`/`-j` flag that prints machine-readable JSON to stdout. The test helpers expect JSON to be present on stdout. However, the CLI emits human-readable log messages via logrus which can appear on stdout as well.

To make tests robust we use a small helper that extracts the JSON object from stdout before attempting to unmarshal. If you prefer stricter separation, consider configuring the logger to write to stderr in the CLI when `--json` is requested.

## Troubleshooting

- If tests fail with JSON unmarshal errors, inspect the captured `errout` (stderr) and `out` values printed by the test failure message. `errout` often contains helpful diagnostics.
- If a contract test reports an unknown command like `unknown command "list" for "message"`, ensure the embedded `messages/` files are present in the checkout. The tests attempt to be resilient to missing message content, but if your environment uses a modified workspace the help/Use strings may differ.
- If builds fail during tests, ensure `GOMODCACHE` and module environment are healthy: `go env` can help diagnose issues.

## Adding tests

- Add unit tests in the package under test.
- Add contract tests to `tests/contract` that build the binary and run it with `runCmd`.

## Example

```bash
cd $(git rev-parse --show-toplevel)
go test ./tests/contract -run TestTagCLI_AddListRemove_JSON -v
```

If you want me to update the CLI to always log to stderr when `--json` is used (so stdout is strictly the JSON payload), say so and I will implement that and update the tests accordingly.
