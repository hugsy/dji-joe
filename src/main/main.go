package main

// +build linux

import (
	// the package
	"dji-joe"

	// standard libraries
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	// external libraries
	"github.com/google/gopacket/pcap"
)

const OUI_CSV_FILE string = "./misc/oui.csv"

var ifaceName = flag.String("i", "", "Specify the interface to read packets from")
var ifaceFromMenu = flag.Bool("l", true, "Choose the interface to read packets from from an interactive menu")
var pcapFileName = flag.String("r", "", "Filename to read from, overrides -i")
var oui_csv_file = flag.String("f", OUI_CSV_FILE, "Path to file holding the MAC prefixes")
var api_endpoint = flag.String("api", "", "URL to the API endpoint")
var verbosity = flag.Int("v", 0, "Verbosity level")
var use5GhzBand = flag.Bool("5", false, "If set, the interface will be scanning the 5GHz band (default: false -> 2.4GHz band)")

/*
Command-line menu to select the network interface if none was provided as an argument.
*/
func ChooseInterface() net.Interface {
	var selectedIface net.Interface

	djijoe.Log.Info("Listing available interfaces")

	ifaces, err := net.Interfaces()
	if err != nil {
		djijoe.Log.FatalF("%+v", err)
	}

	for idx, iface := range ifaces {
		if (iface.Flags & net.FlagLoopback) == 1 {
			continue
		}

		fmt.Printf("[%d] %s (MAC: %s)\n", idx, iface.Name, iface.HardwareAddr)
	}

	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Select interface number: ")
		text, _ := reader.ReadString('\n')
		ifaceIdx, err := strconv.Atoi(strings.TrimSuffix(text, "\n"))
		if err != nil {
			continue
		}

		if (0 <= ifaceIdx) && (ifaceIdx < len(ifaces)) {
			selectedIface = ifaces[ifaceIdx]
			break
		}

		djijoe.Log.WarningF("Incorrect index %d", ifaceIdx)
	}

	return selectedIface
}

/*
Where the magic begins...
*/
func main() {
	var iface net.Interface
	var _iface *net.Interface
	var handle *pcap.Handle
	var err error

	djijoe.Log = djijoe.InitLogger(djijoe.PROGNAME)
	flag.Parse()

	djijoe.Log.InfoF("Starting %s [%s]", djijoe.PROGNAME, djijoe.VERSION)

	if *pcapFileName != "" {
		djijoe.Log.InfoF("From PCAP file: '%s'", *pcapFileName)

		handle, err = pcap.OpenOffline(*pcapFileName)
		if err != nil {
			djijoe.Log.FatalF("PCAP OpenOffline error: %+v", err)
		}

	} else {

		if *ifaceName != "" {
			_iface, err = net.InterfaceByName(*ifaceName)
			if err != nil {
				djijoe.Log.FatalF("%+v", err)
			}
			iface = *_iface
			djijoe.Log.InfoF("Selected interface: '%s'", iface.Name)

		} else {
			djijoe.Log.Info("Selecting interface from menu")
			iface = ChooseInterface()
			djijoe.Log.InfoF("Selected interface: '%s'", iface.Name)
		}

		djijoe.SwitchToModeMonitor(iface)
		defer djijoe.SwitchToModeManaged(iface)

		if *use5GhzBand {
			djijoe.Log.Info("Using 5GHz band")
			err = djijoe.ChangeTo5gBand(iface)
			if err != nil {
				djijoe.Log.FatalF("Failed to change card to 5GHz: %+v", err)
			}
		} else {
			djijoe.Log.Info("Using 2.4GHz band")
			err = djijoe.ChangeTo2gBand(iface)
			if err != nil {
				djijoe.Log.FatalF("Failed to change card to 2GHz: %+v", err)
			}
		}

		go djijoe.ChannelHopper(iface, *use5GhzBand)

		inactive, err := pcap.NewInactiveHandle(iface.Name)
		if err != nil {
			djijoe.Log.FatalF("PCAP NewInactiveHandle error: %+v", err)
		}
		defer inactive.CleanUp()

		inactive.SetSnapLen(1600)
		inactive.SetImmediateMode(true)
		inactive.SetPromisc(true)

		handle, err = inactive.Activate()
		if err != nil {
			djijoe.Log.FatalF("PCAP Activate error: %+v", err)
		}
	}
	defer handle.Close()

	djijoe.Cfg = djijoe.Config{
		Interface:   iface,
		Handle:      handle,
		Vendors:     djijoe.LoadVendorsInfoFromFile(*oui_csv_file),
		ApiEndpoint: *api_endpoint,
		Verbosity:   *verbosity,
	}

	djijoe.DjiGo()
}
