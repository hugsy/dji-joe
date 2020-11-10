#!/bin/bash
#
# Quick install script to have a working Raspbian-based Raspberry-Pi working with dji-joe
#
#
set -e
set -x

test $(id -u) -eq 0 || exit 1

pushd .

# move to a temp dir
dir=$(mktemp -d)
cd ${dir}

# install necessary software
sudo apt update
sudo apt remove avahi* -y
sudo apt install git gcc htop tmux iotop build-essential bc libpcap-dev golang rfkill iw wireless-tools -y

# download raspbian kernel sources, takes some minutes
sudo wget "https://raw.githubusercontent.com/notro/rpi-source/master/rpi-source" -O /usr/bin/rpi-source
sudo chmod 755 /usr/bin/rpi-source
rpi-source

# download the rtl8812au kernel driver and compile it, takes some minutes
git clone --branch 4.3.20 https://github.com/hugsy/rtl8812au_rtl8821au
cd rtl8812au_rtl8821au
make -j4
sudo make install
sudo reboot

popd

exit 0
