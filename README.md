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

## Usage

The dofusdude server is always running with the latest Dofus version and it is highly recommended to use its public endpoints at `https://api.dofusdu.de/`. Try them out [here](https://docs.dofusdu.de) and use the SDKs for real development.

- [Javascript](https://github.com/dofusdude/dofusdude-js) `npm i dofusdude-js --save`
- [Typescript](https://github.com/dofusdude/dofusdude-ts) `npm i dofusdude-ts --save`
- [Go](https://github.com/dofusdude/dodugo) `go get -u github.com/dofusdude/dodugo`
- [Python](https://github.com/dofusdude/dofusdude-py) `pip install dofusdude`
- [Java](https://github.com/dofusdude/dofusdude-java) Maven with GitHub packages setup

If you host your own instance you have to update it yourself. You can use a [doduda Watchdog](https://github.com/dofusdude/doduda#watchdog) as a trigger.

## Development Setup

Assumptions:

- Linux / MacOS

Create a simple `.env` file.

```shell
export MEILI_MASTER_KEY=$(echo $RANDOM | md5sum | head -c 20; echo;)

echo "MEILI_MASTER_KEY=$MEILI_MASTER_KEY" > .env
```

If you had a problem with the md5sum, you are probably on MacOS and need to `brew install md5sha1sum`. Also, `jq` could be a problem, `brew install jq` or `sudo apt install jq`. Or just make up your own keys.

Download [Meilisearch](https://www.meilisearch.com/docs/learn/getting_started/installation#local-installation) for the search engine and let it run in the background.

```shell
curl -L https://install.meilisearch.com | sh
./meilisearch --master-key $MEILI_MASTER_KEY &
```
You can get the process back with `fg` later.

The Almanax data is saved within a file database. `doduapi` can initialize the database structure itself with the following command.
```shell
go run . migrate up
```

You can specify the persistent directory with `--persistent-dir <dir>`. It uses the current working directory per default.

Now build `doduapi`. You need to have [Go](https://go.dev/doc/install) >= 1.18 installed.

```shell
git clone git@github.com:dofusdude/doduapi.git
cd doduapi
go run .
```

If you want more info about the specific tasks, you can set the `LOG_LEVEL` env to one of `debug`, `info`, `warn` (default), `error` or `fatal`. The more left you go in that list, the more info you get.

```bash
LOG_LEVEL=debug go run . --headless
```

## Configuration

Open the `.env` with your favorite editor. Add more parameters if you want. Here is a full list.

```shell
MEILI_MASTER_KEY=<already set> # a random string that must be the same in the meilisearch.service file or parameter
DIR=<working directory> # directory where the ./data dir can be found
DOFUS_VERSION=2.68.5.6 # must match a name from https://github.com/dofusdude/dofus2-main/releases
API_SCHEME=http # http or https. Just used for building links
API_HOSTNAME=localhost # the hostname of the api. Just used for building links
API_PORT=3000 # the port where to listen on
MEILI_PORT=7700 # the port where meilisearch is listening on
MEILI_PROTOCOL=http # http or https
MEILI_HOST=127.0.0.1 # the hostname of meilisearch
PROMETHEUS=false # enable prometheus metrics export running on one apiport + 1
FILESERVER=true # will tell doduapi to serve the image files itself
IS_BETA=false # main (false) vs beta (true)
UPDATE_HOOK_TOKEN=secret # /update/<token> will trigger an update with a POST request {"version": "<dofusversion>"}
```

## Known Problems

Run `doduapi` with `--headless` in a server environment to avoid "no tty" errors.
