<p align="center">
  <img src="https://docs.dofusdu.de/logo_cropped.png" width="120">
  <h3 align="center">doduapi</h3>
  <p align="center">The always up-to-date API for Dofus.</p>
  <p align="center"><a href="https://docs.dofusdu.de">Try it!</a></p>
  <p align="center"><a href="https://goreportcard.com/report/github.com/dofusdude/doduapi"><img src="https://goreportcard.com/badge/github.com/dofusdude/doduapi" alt=""></a> <a href="https://godoc.org/github.com/dofusdude/doduapi"><img src="https://godoc.org/github.com/dofusdude/doduapi?status.svg" alt=""></a> <a href="https://github.com/dofusdude/doduda/actions/workflows/tests.yml"><img src="https://github.com/dofusdude/doduapi/actions/workflows/tests.yml/badge.svg" alt=""></a>
  </p>
</p>

<p align="center">
  <img src="https://vhs.charm.sh/vhs-cPARIJGIyMVFcOYG9b1tZ.gif" width="600">
</p>

## Usage

This project only covers the encyclopedia part.

The dofusdude server is always running with the latest Dofus version and it is highly recommended to use its public endpoint `https://api.dofusdu.de/`. Try out the endpoints [here](https://docs.dofusdu.de) and use the SDKs for real development.

- [Javascript](https://github.com/dofusdude/dofusdude-js) `npm i dofusdude-js --save`
- [Typescript](https://github.com/dofusdude/dofusdude-ts) `npm i dofusdude-ts --save`
- [Go](https://github.com/dofusdude/dodugo) `go get -u github.com/dofusdude/dodugo`
- [Python](https://github.com/dofusdude/dofusdude-py) `pip install dofusdude`
- [PHP](https://github.com/dofusdude/dofusdude-php)

If you host your own instance you have to update it yourself. You can use a [doduda Watchdog](https://github.com/dofusdude/doduda#watchdog) as a trigger.

## Development Setup

Assumptions:
- Linux / MacOS
- Docker (only if `RENDER_IMAGES=true`)

Create a simple `.env` file.
```shell
export MEILI_MASTER_KEY_GEN=$(echo $RANDOM | md5sum | head -c 20; echo;)

echo "DOCKER_MOUNT_DATA_PATH=$(pwd)
MEILI_MASTER_KEY=$MEILI_MASTER_KEY_GEN
CURRENT_UID=$(id -u):$(id -g)
DOFUS_VERSION=$(curl -s https://api.github.com/repos/dofusdude/dofus2-main/releases/latest | jq -r '.name')
" > .env
```
If you had a problem with the md5sum, you are probably on MacOS and need to `brew install md5sha1sum`. Also, `jq` could be a problem, `brew install jq` or `sudo apt install jq`. Or just make up your own keys.

Download [Meilisearch](https://www.meilisearch.com/docs/learn/getting_started/installation#local-installation) for the search engine and let it run in the background.
```shell
curl -L https://install.meilisearch.com | sh
./meilisearch --master-key $MEILI_MASTER_KEY_GEN &
```
You can get the process back with `fg` later.

Now build it from source. You need to have [Go](https://go.dev/doc/install) >= 1.18 installed.
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
DOCKER_MOUNT_DATA_PATH=<already set> # directory where the ./data dir can be found for the image renderer
MEILI_MASTER_KEY=<already set> # a random string that must be the same in the meilisearch.service file or parameter
CURRENT_UID=<already set> # the current user id and group id, used for docker
DOFUS_VERSION=2.68.4.5 # must match a name from https://github.com/dofusdude/dofus2-main/releases
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
RENDER_IMAGES=false # use docker to render images in higher resolutions
```

## Known Problems

If you get some Docker errors and the socket is not at `/var/run/docker.sock`, add a parameter `DOCKER_HOST` to the `.env` file.
```bash
DOCKER_HOST=unix://<your docker.sock path>
```

Run `doduapi` with `--headless` in a server environment to avoid "no tty" errors.
