# Dofusdude - API

An auto-updating API for Dofus by imitating the Ankama Launcher.

See [Docs](https://docs.dofusdu.de) for using this project.

## Dev Setup

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

```shell
echo "DOCKER_MOUNT_DATA_PATH=$(pwd)
MEILI_MASTER_KEY=$(echo $RANDOM | md5sum | head -c 20; echo;)
CURRENT_UID=$(id -u):$(id -g)" > .env
```

Then `sudo docker-compose up`.

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
