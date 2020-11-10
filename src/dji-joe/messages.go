package djijoe

import (
	"net"
	"time"

	"github.com/kellydunn/golang-geo"
)

const (
	TYPE_UNDEFINED     = iota
	TYPE_PROBE_REQUEST = iota
	TYPE_BEACON        = iota
	TYPE_DATA          = iota
)

type HeartBeatMessage struct {
	Timestamp time.Time `json:"ts"`
	Hostname  string    `json:"host"`
}

type WakeUpMessage struct {
	Timestamp time.Time `json:"ts"`
	Hostname  string    `json:"host"`
	Position  geo.Point `json:"position"`
}

type ShutdownMessage struct {
	Timestamp         time.Time `json:"ts"`
	Hostname          string    `json:"host"`
	BeaconFound       uint64    `json:"nb_beacon"`
	ProbeRequestFound uint64    `json:"nb_probes"`
}

type DroneInfoMessage struct {
	Timestamp      time.Time        `json:"ts"`
	Hostname       string           `json:"host"`
	MessageType    int              `json:"type"`
	SignalStrength int8             `json:"strength"`
	Frequency      uint16           `json:"frequency"`
	Vendor         string           `json:"vendor"`
	MacAddress     net.HardwareAddr `json"macaddr"`
}
