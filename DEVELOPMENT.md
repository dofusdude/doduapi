# Developer testing

Run these commands from the `doduapi` repository root.

Run unit tests with `go test -race ./...`. For the HTTP contract suite,
install Docker and run:

```sh
./scripts/test-contract.sh
API_TEST_GAME=dofus3beta ./scripts/test-contract.sh
```

The script creates and removes an isolated Meilisearch container. CI runs the same suite for both
game channels. New routes must have a successful test case to pass the route
coverage guard.

To check a running API against its matching mapped files, use read-only mode:

```sh
DODUAPI_CONTRACT=1 \
API_TEST_BASE_URL=http://localhost:3000 \
DODUDA_OUTPUT_DIR=/absolute/path/to/mapped-data \
go test -count=1 -timeout=30m -run '^TestAPIContract$' -v .
```

Optional settings: `API_TEST_GAME=dofus3beta`, `API_TEST_IMAGES=true`, and
`API_TEST_LANGUAGES=en`. The API must have Almanax initialized. Do not start it
with `--skip-almanax` for a full contract check.

External checks also use `testdata/release-baseline.json` for minimum record
counts and required IDs. After reviewing an intentional data change, update it
with:

```sh
go run ./cmd/api-baseline -data /absolute/path/to/mapped-data -output testdata/release-baseline.json
```

Use `API_TEST_BASELINE` to select a different reviewed baseline. Do not
regenerate baselines automatically in CI.

The query contract snapshot in `testdata/query-contract.json` is checked without
any other repository checkout. To also validate an `api-docs` export, set
`API_DOCS_QUERY_CONTRACT=/absolute/path/to/api-docs/query-contract.json` when
running the tests.
