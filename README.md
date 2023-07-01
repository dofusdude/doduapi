# Dofusdude - API

An auto-updating API for Dofus by imitating the Ankama Launcher.

See [Docs](https://docs.dofusdu.de) for using this project.

## Dev Setup

### MacOS

Install [Homebrew](https://brew.sh/) and [Docker Desktop](https://www.docker.com/products/docker-desktop).
```shell
brew install md5sha1sum
```

We need the Docker socket. Docker Desktop might have a different path to the socket.
If the socket is not at `/var/run/docker.sock`, add a parameter `DOCKER_HOST` to the `.env` file and change the path in the `docker-compose.yml` file.

```yaml
    volumes:
      - ./db:/home/developer/db
      - ./data:/home/developer/data
      - <your docker.sock path>:/var/run/docker.sock
```

```bash
DOCKER_HOST=unix://<your docker.sock path>
```

### Steps

Install Docker and make it available for all users.
```shell
sudo chmod 666 /var/run/docker.sock
```

```shell
git clone git@github.com:dofusdude/api.git
cd api/
git submodule update --init --recursive
sudo docker-compose build

mkdir data
sudo chown -R 1000:1000 data
```

Redis is required for keeping the version state.
```shell
docker-compose -f redis.docker-compose.yml up -d
```

Copy the all lines below together. The newlines are important.
```shell
$ echo "DOCKER_MOUNT_DATA_PATH=$(pwd)
MEILI_MASTER_KEY=$(echo $RANDOM | md5sum | head -c 20; echo;)
CURRENT_UID=$(id -u):$(id -g)" > .env
```

Export the created `.env` to your shell.
You need this for every new shell and if you later use `go run .`.
If you use the entire docker-compose file (unlike specified here), you don't need to export the .env.
```shell
export $(grep -v '^#' .env | xargs)
```

MeiliSearch is required for all searching queries of the API.
```shell
docker-compose up -d meili
```

Start everything. This will take a while.
```shell
go run .
```

For developing, use the following parameters to avoid doing every step again.
- `-clean` Remove all generated and temporary files.
- `-update` Download the Dofus client and generate the `data/` folder.
- `-parse` Parse downloaded data.
- `-gen` Generate in-memory index database.
- `-serve` Start serving the endpoints at the end.

Examples:
```shell
$ go run . -update -parse # Download and generate needed files.
$ go run . -gen -serve # Reuse generated data, generate and run the API.
```

## SWF Renderer
The renderer is my own dockerized gnash image.

If, for some reason, you want to rebuild this image:
```shell
sudo apt install libtool libltdl3-dev autoconf automake pkg-config git build-essential
git clone git://git.sv.gnu.org/gnash.git
cd gnash
./configure
./autogen.sh
sudo ./deb-attempt-install-dependencies.sh
./configure

make
sudo make install
```

Example use of the renderer.
`docker run -v $(pwd):/home/developer --entrypoint /usr/local/bin/dump-gnash dofusdude/swf-renderer --screenshot last --screenshot-file out.png -1 -r1 --width 1000 --height 1000 mount3.swf`
