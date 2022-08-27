# api

Auto generated, always up to date API for all things Dofus.

## Setup

```
git clone git@github.com:dofusdude/api.git
cd api/
git submodule update --init --recursive 
sudo docker-compose build

mkdir data
sudo chown -R 1000:1000 data
```

```
echo "DOCKER_MOUNT_DATA_PATH=$(pwd)
MEILI_MASTER_KEY=$(echo $RANDOM | md5sum | head -c 20; echo;)" > .env
```

`sudo docker-compose up`

## Building the SWF Renderer
sudo apt install libtool libltdl3-dev autoconf automake pkg-config

clone gnash
```
apt install git build-essential
git clone git://git.sv.gnu.org/gnash.git
cd gnash
./configure
./autogen.sh
sudo ./deb-attempt-install-dependencies.sh
./configure

make
sudo make install
```


`docker run -v $(pwd):/home/developer --entrypoint /usr/local/bin/dump-gnash dofusdude/swf-renderer --screenshot last --screenshot-file out.png -1 -r1 --width 1000 --height 1000 mount3.swf`
