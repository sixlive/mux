package audio

/*
#cgo LDFLAGS: -framework CoreAudio -framework AudioToolbox -framework CoreFoundation
#include "audio.h"
*/
import "C"

import (
	"fmt"
	"math"
	"time"
	"unsafe"
)

type Scope int

const (
	ScopeOutput Scope = 0
	ScopeInput  Scope = 1
)

type Device struct {
	Name          string
	UID           string
	TransportType string
	HasInput      bool
	HasOutput     bool
}

type DeviceVolume struct {
	Volume        int
	HasVolume     bool
	AcceptsVolume bool // false if device firmware overrides OS volume changes
}

const maxDevices = 64

func ListDevices() ([]Device, error) {
	var cDevices [maxDevices]C.MuxAudioDevice
	var count C.int

	status := C.mux_get_devices(&cDevices[0], C.int(maxDevices), &count)
	if status != 0 {
		return nil, fmt.Errorf("failed to enumerate audio devices: CoreAudio error %d", status)
	}

	devices := make([]Device, int(count))
	for i := 0; i < int(count); i++ {
		devices[i] = Device{
			Name:          C.GoString(&cDevices[i].name[0]),
			UID:           C.GoString(&cDevices[i].uid[0]),
			TransportType: C.GoString(&cDevices[i].transportType[0]),
			HasInput:      cDevices[i].hasInput != 0,
			HasOutput:     cDevices[i].hasOutput != 0,
		}
	}
	return devices, nil
}

func ListInputDevices() ([]Device, error) {
	all, err := ListDevices()
	if err != nil {
		return nil, err
	}
	var result []Device
	for _, d := range all {
		if d.HasInput {
			result = append(result, d)
		}
	}
	return result, nil
}

func ListOutputDevices() ([]Device, error) {
	all, err := ListDevices()
	if err != nil {
		return nil, err
	}
	var result []Device
	for _, d := range all {
		if d.HasOutput {
			result = append(result, d)
		}
	}
	return result, nil
}

func GetDefaultInputDevice() (*Device, error) {
	uid, err := GetDefaultInputUID()
	if err != nil {
		return nil, err
	}
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.UID == uid {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("default input device not found")
}

func GetDefaultOutputDevice() (*Device, error) {
	uid, err := GetDefaultOutputUID()
	if err != nil {
		return nil, err
	}
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.UID == uid {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("default output device not found")
}

func GetDefaultOutputUID() (string, error) {
	var buf [256]C.char
	status := C.mux_get_default_output_uid(&buf[0], 256)
	if status != 0 {
		return "", fmt.Errorf("failed to get default output device: CoreAudio error %d", status)
	}
	return C.GoString(&buf[0]), nil
}

func GetDefaultInputUID() (string, error) {
	var buf [256]C.char
	status := C.mux_get_default_input_uid(&buf[0], 256)
	if status != 0 {
		return "", fmt.Errorf("failed to get default input device: CoreAudio error %d", status)
	}
	return C.GoString(&buf[0]), nil
}

func SetDefaultInputDevice(uid string) error {
	cUID := C.CString(uid)
	defer C.free(unsafe.Pointer(cUID))
	status := C.mux_set_default_input_device(cUID)
	if status != 0 {
		return fmt.Errorf("failed to set default input device: CoreAudio error %d", status)
	}
	return nil
}

func SetDefaultOutputDevice(uid string) error {
	cUID := C.CString(uid)
	defer C.free(unsafe.Pointer(cUID))
	status := C.mux_set_default_output_device(cUID)
	if status != 0 {
		return fmt.Errorf("failed to set default output device: CoreAudio error %d", status)
	}
	return nil
}

func GetVolume(uid string, scope Scope) (*DeviceVolume, error) {
	cUID := C.CString(uid)
	defer C.free(unsafe.Pointer(cUID))

	var volume C.float
	var hasVolume C.int
	status := C.mux_get_device_volume(cUID, C.int(scope), &volume, &hasVolume)
	if status != 0 {
		return nil, fmt.Errorf("failed to get volume: CoreAudio error %d", status)
	}

	return &DeviceVolume{
		Volume:    int(math.Round(float64(volume) * 100)),
		HasVolume: hasVolume != 0,
	}, nil
}

func SetVolume(uid string, scope Scope, volume int) error {
	cUID := C.CString(uid)
	defer C.free(unsafe.Pointer(cUID))

	fVol := C.float(float32(volume) / 100.0)
	status := C.mux_set_device_volume(cUID, C.int(scope), fVol)
	if status != 0 {
		return fmt.Errorf("failed to set volume: CoreAudio error %d", status)
	}
	return nil
}

// ProbeVolumeControl checks if a device actually accepts OS-level volume changes.
// Some devices (e.g. Shure MV7+) have firmware-controlled gain that overrides
// any changes made through CoreAudio. This function sets a test value, waits
// for the device to settle, and checks if the change persisted.
func ProbeVolumeControl(uid string, scope Scope) *DeviceVolume {
	vol, err := GetVolume(uid, scope)
	if err != nil || !vol.HasVolume {
		return &DeviceVolume{HasVolume: false}
	}

	original := vol.Volume

	// Pick a test value different from current
	testVal := original + 10
	if testVal > 100 {
		testVal = original - 10
	}

	_ = SetVolume(uid, scope, testVal)
	time.Sleep(150 * time.Millisecond)

	after, err := GetVolume(uid, scope)
	if err != nil {
		return &DeviceVolume{Volume: original, HasVolume: true, AcceptsVolume: false}
	}

	// Restore original
	_ = SetVolume(uid, scope, original)

	diff := after.Volume - testVal
	if diff < 0 {
		diff = -diff
	}
	accepts := diff <= 3

	return &DeviceVolume{Volume: original, HasVolume: true, AcceptsVolume: accepts}
}

func GetVolumeMethod(uid string, scope Scope) string {
	cUID := C.CString(uid)
	defer C.free(unsafe.Pointer(cUID))
	method := C.mux_get_volume_method(cUID, C.int(scope))
	switch method {
	case 0:
		return "none"
	case 1:
		return "hal-master"
	case 2:
		return "hal-channels"
	case 3:
		return "service"
	default:
		return fmt.Sprintf("unknown(%d)", method)
	}
}
