package djijoe

import (
	"errors"
	"net"
	"syscall"
	"time"
	"unsafe"
)

const (
	SIOCGIWNAME = 0x8B01
	SIOCSIWMODE = 0x8B06
	SIOCGIWMODE = 0x8B07
	SIOCSIWFREQ = 0x8B04

	// http://elixir.free-electrons.com/linux/v4.11.5/source/include/uapi/linux/wireless.h#L478
	IW_MODE_MONITOR = 0x06
	IW_MODE_MANAGED = 0x02

	IW_FREQ_FIXED = 0x01
)

// http://elixir.free-electrons.com/linux/v4.11.5/source/include/uapi/linux/if.h#L241
type ifreq struct {
	name  [syscall.IFNAMSIZ]byte
	flags uint16
}

// http://elixir.free-electrons.com/linux/v4.11.5/source/include/uapi/linux/wireless.h#L980
type iwreq struct {
	name  [syscall.IFNAMSIZ]byte
	flags uint32
}

// http://elixir.free-electrons.com/linux/v4.11.5/source/include/uapi/linux/wireless.h#L736
type iwfreq struct {
	name  [syscall.IFNAMSIZ]byte
	m     int32
	e     int16
	i     uint8
	flags uint8
}

type Frequency struct {
	mantissa int
	exponent int
	channel  int
}

type Frequencies []Frequency

// Valid WiFi frequencies
// 2.4GHz band
var Wifi2GzFrequencies = Frequencies{
	Frequency{exponent: 6, mantissa: 2412, channel: 1},
	Frequency{exponent: 6, mantissa: 2417, channel: 2},
	Frequency{exponent: 6, mantissa: 2422, channel: 3},
	Frequency{exponent: 6, mantissa: 2427, channel: 4},
	Frequency{exponent: 6, mantissa: 2432, channel: 5},
	Frequency{exponent: 6, mantissa: 2437, channel: 6},
	Frequency{exponent: 6, mantissa: 2442, channel: 7},
	Frequency{exponent: 6, mantissa: 2447, channel: 8},
	Frequency{exponent: 6, mantissa: 2452, channel: 9},
	Frequency{exponent: 6, mantissa: 2457, channel: 10},
	Frequency{exponent: 6, mantissa: 2462, channel: 11},
	Frequency{exponent: 6, mantissa: 2467, channel: 12},
	Frequency{exponent: 6, mantissa: 2472, channel: 13},
}

// 2.4GHz band
var Wifi5GzFrequencies = Frequencies{
	Frequency{exponent: 7, mantissa: 518, channel: 36},
	Frequency{exponent: 7, mantissa: 520, channel: 40},
	Frequency{exponent: 7, mantissa: 522, channel: 44},
	Frequency{exponent: 7, mantissa: 524, channel: 48},
	Frequency{exponent: 7, mantissa: 526, channel: 52},
	Frequency{exponent: 7, mantissa: 528, channel: 56},
	Frequency{exponent: 7, mantissa: 530, channel: 60},
	Frequency{exponent: 7, mantissa: 532, channel: 64},
	Frequency{exponent: 7, mantissa: 550, channel: 100},
	Frequency{exponent: 7, mantissa: 552, channel: 104},
	Frequency{exponent: 7, mantissa: 554, channel: 108},
	Frequency{exponent: 7, mantissa: 556, channel: 112},
	Frequency{exponent: 7, mantissa: 558, channel: 116},
	Frequency{exponent: 7, mantissa: 560, channel: 120},
	Frequency{exponent: 7, mantissa: 562, channel: 124},
	Frequency{exponent: 7, mantissa: 564, channel: 128},
	Frequency{exponent: 7, mantissa: 566, channel: 132},
	Frequency{exponent: 7, mantissa: 568, channel: 136},
	Frequency{exponent: 7, mantissa: 570, channel: 140},
}

/*
Change the state of the interface via ioctl: if `setUp` is true, then this is
equivalent to `ifup <ifname>`.
*/
func SetInterfaceUp(name string, setUp bool) error {

	sockFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(sockFd)

	var ifl ifreq
	copy(ifl.name[:], []byte(name))

	// retrieve the current flags for the interface
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sockFd), syscall.SIOCGIFFLAGS,
		uintptr(unsafe.Pointer(&ifl)))
	if errno != 0 {
		err = errno
		Log.ErrorF("ioctl(SIOCSIFFLAGS) failed: %+v", err)
		return err
	}

	if setUp {
		ifl.flags |= uint16(syscall.IFF_UP)
	} else {
		ifl.flags &^= uint16(syscall.IFF_UP)
	}

	// apply the new set of flags
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(sockFd), syscall.SIOCSIFFLAGS,
		uintptr(unsafe.Pointer(&ifl)))
	if errno != 0 {
		err = errno
		Log.ErrorF("ioctl(SIOCSIFFLAGS) failed: %+v", err)
		return err
	}
	return nil
}

/*

 */
func ioctlSetMode(ifName string, mode int) error {

	sockFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_IP)
	if err != nil {
		Log.ErrorF("Socket() failed: %+v", err)
		return err
	}
	defer syscall.Close(sockFd)

	var iwl iwreq
	copy(iwl.name[:], []byte(ifName))
	iwl.flags = uint32(mode)

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sockFd), SIOCSIWMODE,
		uintptr(unsafe.Pointer(&iwl)))
	if errno != 0 {
		err = errno
		Log.ErrorF("ioctl(SIOCSIWMODE) failed: %+v", err)
		return err
	}

	return nil
}

/*
Change the selected interface mode.

*/
func SwitchToMode(iface net.Interface, mode int) error {
	var mode_str string
	var err error

	switch mode {
	case IW_MODE_MONITOR:
		Log.DebugF("Switching '%s' ->  IW_MODE_MONITOR", iface.Name)
		mode_str = "Monitor"
	case IW_MODE_MANAGED:
		Log.DebugF("Switching '%s' ->  IW_MODE_MANAGED", iface.Name)
		mode_str = "Managed"
	default:
		return errors.New("Incorrect mode")
	}

	err = SetInterfaceUp(iface.Name, false)
	if err != nil {
		Log.FatalF("Failed to set '%s' state to up: %+v", iface.Name, err)
	}
	Log.DebugF("'%s' is down", iface.Name)

	err = ioctlSetMode(iface.Name, mode)
	if err != nil {
		Log.FatalF("Failed to switch '%s' to mode '%s': %+v", iface.Name, mode_str, err)
	}
	Log.InfoF("'%s' new mode: %s", iface.Name, mode_str)

	err = SetInterfaceUp(iface.Name, true)
	if err != nil {
		Log.FatalF("Failed to set '%s' state to up: %+v", iface.Name, err)
	}
	Log.DebugF("'%s' is back up", iface.Name)

	return nil
}

func SwitchToModeMonitor(iface net.Interface) error {
	return SwitchToMode(iface, IW_MODE_MONITOR)
}

func SwitchToModeManaged(iface net.Interface) error {
	return SwitchToMode(iface, IW_MODE_MANAGED)
}

/*
Change 802.11 channel
*/
func ChangeChannel(iface net.Interface, freq Frequency) error {

	sockFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_IP)
	if err != nil {
		Log.ErrorF("[ChangeChannel]Socket() failed: %+v", err)
		return err
	}
	defer syscall.Close(sockFd)

	var iwf iwfreq
	copy(iwf.name[:], []byte(iface.Name))
	iwf.m = int32(freq.channel)
	iwf.e = int16(0)
	iwf.flags = uint8(IW_FREQ_FIXED)

	if Cfg.Verbosity > 2 {
		Log.DebugF("Setting frequency=%d.1e%dGHz (channel=%d)",
			freq.mantissa, freq.exponent, freq.channel)
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sockFd), SIOCSIWFREQ,
		uintptr(unsafe.Pointer(&iwf)))
	if errno != 0 {
		err = errno
		Log.ErrorF("ioctl(SIOCSIWFREQ) failed: %+v", err)
		return err
	}

	return nil
}

/*
GoRoutine for channel hopping
*/
func ChannelHopper(iface net.Interface, use5GhzBand bool) {
	var channels Frequencies

	if use5GhzBand {
		channels = Wifi5GzFrequencies
	} else {
		channels = Wifi2GzFrequencies
	}
	var i int = 0
	var interval time.Duration = 500 * time.Millisecond

	for {
		err := ChangeChannel(iface, channels[i])
		if err != nil {
			break
		}
		i = (i + 1) % len(channels)
		time.Sleep(interval)
	}
}

/*
Set the initial WiFi frequency.
*/
func ChangeTo5gBand(iface net.Interface) error {
	return ChangeChannel(iface, Wifi5GzFrequencies[0])
}

func ChangeTo2gBand(iface net.Interface) error {
	return ChangeChannel(iface, Wifi2GzFrequencies[0])
}
