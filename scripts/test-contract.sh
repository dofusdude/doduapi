#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# Never use the developer's normal search server. This container has no volume,
# binds only loopback, uses a random free port, and is removed on exit.
container_id=$(docker run --detach --rm --publish 127.0.0.1::7700 \
  --env MEILI_MASTER_KEY=contract-test-key \
  --env MEILI_NO_ANALYTICS=true getmeili/meilisearch:v1.12)
trap 'docker stop "$container_id" >/dev/null 2>&1 || true' EXIT
address=$(docker port "$container_id" 7700/tcp)
export DODUAPI_TEST_MEILI_URL="http://$address"
export DODUAPI_TEST_MEILI_KEY=contract-test-key
export DODUAPI_CONTRACT=1
unset API_TEST_BASE_URL
for attempt in $(seq 1 30); do
  if curl --fail --silent "$DODUAPI_TEST_MEILI_URL/health" >/dev/null; then
    report=$(mktemp "${TMPDIR:-/tmp}/doduapi-contract.XXXXXX")
    if GOWORK=off go test -race -count=1 -timeout=15m -run '^TestAPIContract$' -json . > "$report"; then
      echo "API contract passed. JSON report: $report"
      exit 0
    fi
    echo "API contract FAILED. JSON report: $report" >&2
    awk '/"Action":"fail"|"OutputType":"error"/' "$report"
    exit 1
  fi
  sleep 1
done
echo 'Test search server did not become ready.' >&2
exit 1
