package djijoe

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/kellydunn/golang-geo"
)

const (
	PROBE_STATE_AWAKE    = iota
	PROBE_STATE_RUNNING  = iota
	PROBE_STATE_SHUTDOWN = iota
	PROBE_STATE_STOPPED  = iota
)

type Probe struct {
	State            int
	StartTime        time.Time
	EndTime          time.Time
	LastHeartbeat    time.Time
	Hostname         string
	EnableApi        bool
	ApiEndpoint      url.URL
	NbBeacons        uint64
	NbProbes         uint64
	NbBytesCollected uint64
	GpsCoordinates   geo.Point
}

type Probes []Probe

func (p *Probe) GetUrlTo(Path string) string {
	return fmt.Sprintf("%s://%s%s",
		p.ApiEndpoint.Scheme,
		p.ApiEndpoint.Host,
		Path)
}

func (p *Probe) NotifyWakeup() bool {
	if !p.EnableApi {
		return false
	}

	var msg = WakeUpMessage{
		Hostname:  p.Hostname,
		Timestamp: p.StartTime,
	}
	Log.DebugF("Sending WAKEUP from %s at %s", p.Hostname, p.StartTime)
	jsonValue, err := json.Marshal(msg)
	if err != nil {
		return false
	}

	Url := p.GetUrlTo(API_WAKEUP)
	resp, err := http.Post(Url, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		Log.ErrorF("NotifyWakeup() HTTP POST failed: %+v", err)
		p.EnableApi = false
		return false
	}
	return resp.StatusCode == http.StatusNoContent
}

func (p *Probe) NotifyShutdown() bool {
	if !p.EnableApi {
		return false
	}

	var msg = ShutdownMessage{
		Hostname:          p.Hostname,
		Timestamp:         p.EndTime,
		BeaconFound:       p.NbBeacons,
		ProbeRequestFound: p.NbProbes,
	}

	Log.DebugF("Sending SHUTDOWN from %s at %s", p.Hostname, p.EndTime)
	jsonValue, err := json.Marshal(msg)
	if err != nil {
		return false
	}

	Url := p.GetUrlTo(API_SHUTDOWN)
	resp, err := http.Post(Url, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		Log.ErrorF("NotifyShutdown() HTTP POST failed: %+v", err)
		p.EnableApi = false
		return false
	}
	return resp.StatusCode == http.StatusNoContent
}

func (p *Probe) ProcessFlaggedPacket(info DroneInfoMessage) error {
	if !p.EnableApi {
		return nil
	}

	var httpTransport = &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}

	var httpRequest = &http.Client{
		Timeout:   10 * time.Second,
		Transport: httpTransport,
	}

	jsonValue, err := json.Marshal(info)
	if err != nil {
		Log.ErrorF("ProcessFlaggedPacket(): JSON Marshalling failed: %+v", err)
		return err
	}

	Url := p.GetUrlTo(API_NEWDRONEINFO)
	httpResponse, err := httpRequest.Post(Url, "application/json", bytes.NewBuffer(jsonValue))
	if err != nil {
		Log.ErrorF("NEWDRONEINFO HTTP request failed: %+v", err)
		p.EnableApi = false
		return err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusAccepted {
		Log.ErrorF("Unexpected response: got %d , expected %d", httpResponse.StatusCode, http.StatusOK)
		return nil
	}

	return nil
}

func (p *Probe) Wakeup() error {
	var interval time.Duration = 30 * time.Second

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "NONAME"
	}

	p.StartTime = time.Now()
	p.State = PROBE_STATE_AWAKE
	p.StartTime = time.Now()
	p.Hostname = hostname
	p.NbBeacons = uint64(0)
	p.NbProbes = uint64(0)
	p.NbBytesCollected = uint64(0)

	Log.DebugF("Starting probe '%s'", p.Hostname)

	// notify server of new probe
	p.NotifyWakeup()

	// and start the Heartbeat goroutine
	p.State = PROBE_STATE_RUNNING
	go p.SendHeartbeat(interval)
	return nil
}

func (p *Probe) SendHeartbeat(interval time.Duration) {
	if !p.EnableApi {
		return
	}

	for {
		if p.State != PROBE_STATE_RUNNING {
			break
		}
		time.Sleep(interval)

		var msg = HeartBeatMessage{
			Hostname:  p.Hostname,
			Timestamp: time.Now(),
		}

		Log.DebugF("Sending HEARTBEAT from %s", p.Hostname)
		jsonValue, err := json.Marshal(msg)
		if err != nil {
			break
		}

		Url := p.GetUrlTo(API_HEARTBEAT)
		_, err = http.Post(Url, "application/json", bytes.NewBuffer(jsonValue))
		if err != nil {
			Log.ErrorF("SendHeartbeat() HTTP POST failed: %+v", err)
			p.EnableApi = false
			break
		}
	}
}

func (p *Probe) Shutdown() error {
	Log.DebugF("Stopping probe '%s'", p.Hostname)

	p.EndTime = time.Now()
	p.State = PROBE_STATE_STOPPED

	Log.InfoF("Finished monitoring in %d ms, read %d bytes",
		(p.EndTime.UnixNano()-p.StartTime.UnixNano())/1000, p.NbBytesCollected)
	Log.InfoF("Discovered %d DJI ProbeRequests, %d DJI Beacon", p.NbProbes, p.NbBeacons)

	// notify server of shutdown
	p.NotifyShutdown()
	return nil
}

func (p *Probe) SetApiEndpoint(ApiEndpoint string) error {
	u, err := url.Parse(ApiEndpoint)
	if err != nil {
		Log.DebugF("Refusing change of API Endpoint to '%s': invalid URL: %+v", ApiEndpoint, err)
		p.EnableApi = false
		return err
	}

	Log.DebugF("Changing API Endpoint to '%s'", ApiEndpoint)
	p.ApiEndpoint = *u
	p.EnableApi = true
	return nil
}

func (p *Probe) SetGpsCoordinates(lat float64, long float64) error {
	Log.DebugF("Updating GPS position of '%s' to (%.5f, %.5f)", p.Hostname, lat, long)
	pt := geo.NewPoint(lat, long)
	p.GpsCoordinates = *pt
	return nil
}

func (p Probe) String() string {
	return fmt.Sprintf("<Probe name='%s'>", p.Hostname)
}

func GetProbeByName(p Probes, name string) (int, *Probe) {
	for i, curp := range p {
		if curp.Hostname == name {
			return i, &curp
		}
	}

	return -1, nil
}

func RemoveProbeByName(p Probes, name string) (Probes, error) {
	idx, _ := GetProbeByName(p, name)
	if idx == -1 {
		return p, errors.New("Invalid Probe")
	}

	return append(p[:idx], p[idx+1:]...), nil
}

func RemoveProbe(p Probes, _p *Probe) (Probes, error) {
	return RemoveProbeByName(p, _p.Hostname)
}
