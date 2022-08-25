# api

Auto generated, always up to date API for all things Dofus.

sudo apt-get install zlib1g zlib1g-dev libjpeg-dev

sudo apt install autoconf automake gconf-2.0 pkg-config libltdl3-dev build-essential libgconf2-dev

libtool libltdl3-dev autoconf automake pkg-config

sudo ./deb-attempt-install-dependencies.sh

in gnash
./autogen.sh
./configure
make
sudo make install

dump-gnash --screenshot last --screenshot-file out.png -1 -r1 --width 1000 --height 1000 mount3.swf

apt install git build-essential
git clone git://git.sv.gnu.org/gnash.git
cd gnash
./configure
./autogen.sh
sudo ./deb-attempt-install-dependencies.sh

docker run -v $(pwd):/home/developer --entrypoint /usr/local/bin/dump-gnash dofusdude/swf-renderer --screenshot last --screenshot-file out.png -1 -r1 --width 1000 --height 1000 mount3.swf
