## DJI-Joe ##

WiFi scanner for drones.

**Note** : DJI-Joe only works on WiFi cards
allowing [Monitor mode](https://en.wikipedia.org/wiki/Monitor_mode).

DJI-Joe will setup the given interface to Monitor mode, and look up for possible
drones based on the MAC address prefixes. The known prefixes are loaded from a
CSV file, by default at `misc/oui.csv`. Once loaded, DJI-Joe will only notify the
presence of 802.11i Beacon (sent by remote) or ProbeRequest (sent by UAV) packets.


### Compilation

#### Native compilation

The project uses [PCAP library](https://github.com/the-tcpdump-group/libpcap):
```
$ sudo apt install libpcap-dev
or
$ sudo yum install libpcap-devel
```

Use GoLang compiler `go-build`, and from the project root, type:

```
$ sudo apt install golang
$ export GOPATH=${GOPATH}:/path/to/dji-joe
$ go get github.com/apsdehal/go-logger
$ go get github.com/google/gopacket
$ go build -o bin/dji-joe-$(arch) src/main/main.go
```

#### Cross-compilation

To cross-compile to another architecture, simply use the `GOOS` and `GOARCH`
environment variables. For ARM (i.e. Raspberry-Pi), Go supports
several [EABI](https://github.com/golang/go/wiki/GoArm) so you can cross-compile
with:

```
$ GOARCH=arm GOARM=7 CGO_ENABLED=1 CGO_LDFLAGS+="-g -O2 -L$(pwd)/misc -lpcap" CC=arm-linux-gnueabi-gcc go build -o bin/dji-joe-arm src/main/main.go
```


#### Raspberry-Pi

##### Update your Raspbian

```
$ for i in update upgrade autoclean autoremove ; do sudo apt $i -y ;done && sudo reboot
```

Then install all the pre-requisites by running:

```
$ sudo /path/to/dji-joe/misc/install-rtl8812au-rpi.sh
```

You might go grab a coffee as this script may take a while to finish.


##### (Re-)Building the AWUS036AC manually

If you need to, you can (re)compile the working RTL8812AU driver for the Alfa
AWUS036AC to support Monitor mode:

```
git clone https://github.com/hugsy/rtl8812au_rtl8821au
cd rtl8812au_rtl8821au
make -j4 && sudo make install
sudo reboot
```

Tested with:

 - [Raspbian](https://www.raspberrypi.org/downloads/raspbian/) **prefered**
 - [Ubuntu](https://wiki.ubuntu.com/ARM/RaspberryPi)



### Runtime

_Note_: DJI-Joe requires root privilege to change the WiFi card to Monitor mode.

If not compiled:

```
$ sudo go run ./src/main/main.go -i wlan0
```

Else:

```
$ sudo bin/dji-joe -i wlan0
```

Which should execute and show something similar to the following:

```
$ sudo ./bin/dji-joe -i wlp1s0
#1 2017-06-09 23:52:50 main.go:144 ▶ INF Starting DJI-Joe [0.0.1]
#2 2017-06-09 23:52:50 main.go:162 ▶ INF Selected interface: 'wlp1s0'
#3 2017-06-09 23:52:50 network.go:115 ▶ DEB Switching 'wlp1s0' ->  IW_MODE_MONITOR
#4 2017-06-09 23:52:50 network.go:128 ▶ DEB 'wlp1s0' is down
#5 2017-06-09 23:52:50 network.go:134 ▶ INF 'wlp1s0' new mode: Monitor
#6 2017-06-09 23:52:50 network.go:140 ▶ DEB 'wlp1s0' is back up
#7 2017-06-09 23:52:50 main.go:123 ▶ INF Added prefix '60601f' for vendor 'DJI'
#8 2017-06-09 23:52:50 main.go:123 ▶ INF Added prefix '903ae6' for vendor 'Parrot SA'
#9 2017-06-09 23:52:50 main.go:123 ▶ INF Added prefix '9003b7' for vendor 'Parrot SA'
#10 2017-06-09 23:52:50 main.go:123 ▶ INF Added prefix 'a0143d' for vendor 'Parrot SA'
#11 2017-06-09 23:52:50 main.go:123 ▶ INF Added prefix '00267c' for vendor 'Parrot SA'
#12 2017-06-09 23:52:50 main.go:123 ▶ INF Added prefix '00121c' for vendor 'Parrot SA'
#13 2017-06-09 23:52:50 djijoe.go:63 ▶ INF Starting to read packets
#14 2017-06-09 23:53:05 djijoe.go:116 ▶ NOT Found 802.11 ProbeRequest from vendor DJI (device 60601f4211b8) - strength=-26 dBm
#15 2017-06-09 23:53:05 djijoe.go:116 ▶ NOT Found 802.11 ProbeRequest from vendor DJI (device 60601f4211b8) - strength=-27 dBm
#16 2017-06-09 23:53:08 djijoe.go:116 ▶ NOT Found 802.11 ProbeRequest from vendor DJI (device 60601f4211b8) - strength=-26 dBm
#17 2017-06-09 23:53:11 djijoe.go:116 ▶ NOT Found 802.11 ProbeRequest from vendor DJI (device 60601f4211b8) - strength=-32 dBm
#18 2017-06-09 23:53:11 djijoe.go:116 ▶ NOT Found 802.11 ProbeRequest from vendor DJI (device 60601f4211b8) - strength=-34 dBm
#19 2017-06-09 23:53:16 djijoe.go:116 ▶ NOT Found 802.11 ProbeRequest from vendor DJI (device 60601f4211b8) - strength=-32 dBm
```

If the `--api` is provided on the command line, with a valid HTTP URL option,
DJI-Joe will push all the detection events to the
server [`DJI-Jane`](https://github.com/hugsy/dji-jane).

### Add new drone MAC to the signature database

Simply add a 2 field CSV entry to `misc/oui.csv` , where

 - `field1` : vendor name
 - `field2` : hexadecimal representation of the MAC address

 For example, to add `NewDroneVendor` whose MAC prefix is 00:aa:ff, simply do:

 ```
 $ echo 'NewDroneVendor;00aaff' >> /path/to/oui.csv
 ```
