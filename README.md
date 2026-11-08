<p align="center">
  <img src="https://docs.dofusdu.de/dofus2/logo_cropped.png" width="120">
  <h3 align="center">doduapi</h3>
  <p align="center">Open Dofus Encyclopedia API</p>
  <p align="center"><a href="https://docs.dofusdu.de">Try it!</a></p>
  <p align="center"><a href="https://goreportcard.com/report/github.com/dofusdude/doduapi"><img src="https://goreportcard.com/badge/github.com/dofusdude/doduapi" alt=""></a> <a href="https://github.com/dofusdude/doduda/actions/workflows/tests.yml"><img src="https://github.com/dofusdude/doduapi/actions/workflows/tests.yml/badge.svg" alt=""></a>
  </p>
</p>

<p align="center">
  <img src="https://vhs.charm.sh/vhs-2mgsbcqX7zIII0IvqV5uw0.gif" width="600">
</p>

## Public Instance

The dofusdude server is always running with the latest Dofus version and it is highly recommended to use its public endpoints at `https://api.dofusdu.de/`. Try them out [here](https://docs.dofusdu.de) and use the SDKs for real development.

- [Javascript](https://github.com/dofusdude/dofusdude-js) `npm i dofusdude-js --save`
- [Typescript](https://github.com/dofusdude/dofusdude-ts) `npm i dofusdude-ts --save`
- [Go](https://github.com/dofusdude/dodugo) `go get -u github.com/dofusdude/dodugo`
- [Python](https://github.com/dofusdude/dofusdude-py) `pip install dofusdude`
- [Java](https://github.com/dofusdude/dofusdude-java) Maven with GitHub packages setup

## Self-Hosting

If you want to host `doduapi` yourself, just follow these commands.

```shell
# Prepare search engine environment
export MEILI_MASTER_KEY=supersecretandhighlysecurekey
echo "MEILI_MASTER_KEY=$MEILI_MASTER_KEY" > .env

# Install and run search engine
curl -L https://install.meilisearch.com | sh
./meilisearch --master-key $MEILI_MASTER_KEY &

# Install and run doduapi
curl -s https://get.dofusdu.de/doduapi | sh
doduapi migrate up
doduapi
```
You can get the search engine process back with `fg` later.

## Configuration

Open the `.env` with your favorite editor. Add more parameters if you want. Here is a full list.

```shell
MEILI_MASTER_KEY=<already set> # a random string that must be the same in the meilisearch.service file or parameter
DIR=<working directory> # directory where the ./data dir can be found
DOFUS_VERSION=3.0.40.28 # must match a name from https://github.com/dofusdude/dofus3-main/releases
DEVMODE=false # use locally generated doduda files instead of GitHub release assets
DODUDA_OUTPUT_DIR=/absolute/path/to/mapped-data # required in devmode; directory containing MAPPED_*.json
API_SCHEME=http # http or https. Just used for building links
API_HOSTNAME=localhost # the hostname of the api. Just used for building links
API_PORT=3000 # the port where to listen on
MEILI_PORT=7700 # the port where meilisearch is listening on
MEILI_PROTOCOL=http # http or https
MEILI_HOST=127.0.0.1 # the hostname of meilisearch
PROMETHEUS=false # enable prometheus metrics export running on one apiport + 1
FILESERVER=true # will tell doduapi to serve the image files itself
ALMANAX_MAX_LOOKAHEAD_DAYS=365 # maximum date range size
ALMANAX_DEFAULT_LOOKAHEAD_DAYS=6 # default date range size
IS_BETA=false # main (false) vs beta (true)
UPDATE_HOOK_TOKEN=secret # /update/<token> will trigger an update with a POST request {"version": "<dofusversion>"}
```

When `DEVMODE=true`, set `DODUDA_OUTPUT_DIR` to your mapped-data directory.
`DOFUS_VERSION` is optional and image downloads are skipped. Keep doduda's
output layout: mapped JSON and `img/` inside that directory, with element and
item-type catalogs in `persistent/` beside it. No other repository checkout
is required. Relative paths are resolved from the current working directory.

## JSON queries

Collection and search endpoints support HTTP `QUERY` with an `application/json`
body. See the [API specification](https://github.com/dofusdude/api-docs/blob/main/openapi-3.2.yaml)
for supported filters, fields, includes, and limits for each endpoint.

```sh
curl -X QUERY 'http://localhost:3000/dofus3/v1/en/items/equipment' \
  -H 'Content-Type: application/json' \
  --data '{"filter":{"min_level":100,"type":{"name_id":["hat"]}},"fields":{"item":["effects"]},"sort":["-level"],"page":{"size":20}}'
```

## Tests

See the [developer testing guide](DEVELOPMENT.md) for local and CI tests.

## Updating API documentation

Endpoint parameters and expansion fields are declared in `query/contract.go`.
After changing the registry, regenerate the local contract snapshot and export
the docs contract to an explicitly selected `api-docs` checkout:

```sh
go run ./cmd/query-contract -output testdata/query-contract.json
API_DOCS_DIR=/absolute/path/to/api-docs
go run ./cmd/query-contract -output "$API_DOCS_DIR/query-contract.json"
npm --prefix "$API_DOCS_DIR" ci
npm --prefix "$API_DOCS_DIR" run build
```

## Known Problems

Run `doduapi` with `--headless` in a server environment to avoid "no tty" errors.
