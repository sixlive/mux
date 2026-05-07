#include "audio.h"
#include <AudioToolbox/AudioServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

static int cfstring_to_cstr(CFStringRef cfStr, char *buf, int bufLen) {
    if (cfStr == NULL) {
        buf[0] = '\0';
        return -1;
    }
    Boolean ok = CFStringGetCString(cfStr, buf, bufLen, kCFStringEncodingUTF8);
    return ok ? 0 : -1;
}

static void transport_type_str(UInt32 type, char *buf, int bufLen) {
    switch (type) {
    case kAudioDeviceTransportTypeBuiltIn:
        strncpy(buf, "built-in", bufLen);
        break;
    case kAudioDeviceTransportTypeUSB:
        strncpy(buf, "usb", bufLen);
        break;
    case kAudioDeviceTransportTypeBluetooth:
        strncpy(buf, "bluetooth", bufLen);
        break;
    case kAudioDeviceTransportTypeBluetoothLE:
        strncpy(buf, "bluetooth-le", bufLen);
        break;
    case kAudioDeviceTransportTypeHDMI:
        strncpy(buf, "hdmi", bufLen);
        break;
    case kAudioDeviceTransportTypeDisplayPort:
        strncpy(buf, "displayport", bufLen);
        break;
    case kAudioDeviceTransportTypeAirPlay:
        strncpy(buf, "airplay", bufLen);
        break;
    case kAudioDeviceTransportTypeThunderbolt:
        strncpy(buf, "thunderbolt", bufLen);
        break;
    case kAudioDeviceTransportTypeVirtual:
        strncpy(buf, "virtual", bufLen);
        break;
    case kAudioDeviceTransportTypeAggregate:
        strncpy(buf, "aggregate", bufLen);
        break;
    default:
        strncpy(buf, "unknown", bufLen);
        break;
    }
    buf[bufLen - 1] = '\0';
}

static int device_has_streams(AudioObjectID deviceID, AudioObjectPropertyScope scope) {
    AudioObjectPropertyAddress addr = {
        kAudioDevicePropertyStreams,
        scope,
        kAudioObjectPropertyElementMain
    };
    UInt32 size = 0;
    OSStatus status = AudioObjectGetPropertyDataSize(deviceID, &addr, 0, NULL, &size);
    if (status != noErr) return 0;
    return (size / sizeof(AudioStreamID)) > 0;
}

static AudioObjectID find_device_by_uid(const char *uid) {
    AudioObjectPropertyAddress addr = {
        kAudioHardwarePropertyTranslateUIDToDevice,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    CFStringRef uidStr = CFStringCreateWithCString(NULL, uid, kCFStringEncodingUTF8);
    if (uidStr == NULL) return kAudioObjectUnknown;

    AudioObjectID deviceID = kAudioObjectUnknown;
    UInt32 size = sizeof(deviceID);
    OSStatus status = AudioObjectGetPropertyData(
        kAudioObjectSystemObject, &addr,
        sizeof(uidStr), &uidStr, &size, &deviceID);
    CFRelease(uidStr);

    if (status != noErr) return kAudioObjectUnknown;
    return deviceID;
}

int mux_get_devices(MuxAudioDevice *devices, int maxDevices, int *outCount) {
    AudioObjectPropertyAddress addr = {
        kAudioHardwarePropertyDevices,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };

    UInt32 size = 0;
    OSStatus status = AudioObjectGetPropertyDataSize(
        kAudioObjectSystemObject, &addr, 0, NULL, &size);
    if (status != noErr) return status;

    int count = size / sizeof(AudioObjectID);
    if (count > maxDevices) count = maxDevices;

    AudioObjectID deviceIDs[count];
    size = count * sizeof(AudioObjectID);
    status = AudioObjectGetPropertyData(
        kAudioObjectSystemObject, &addr, 0, NULL, &size, deviceIDs);
    if (status != noErr) return status;

    count = size / sizeof(AudioObjectID);

    for (int i = 0; i < count; i++) {
        memset(&devices[i], 0, sizeof(MuxAudioDevice));
        devices[i].deviceID = deviceIDs[i];

        // Name
        AudioObjectPropertyAddress nameAddr = {
            kAudioObjectPropertyName,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        CFStringRef nameRef = NULL;
        UInt32 nameSize = sizeof(nameRef);
        status = AudioObjectGetPropertyData(deviceIDs[i], &nameAddr, 0, NULL, &nameSize, &nameRef);
        if (status == noErr && nameRef != NULL) {
            cfstring_to_cstr(nameRef, devices[i].name, sizeof(devices[i].name));
            CFRelease(nameRef);
        }

        // UID
        AudioObjectPropertyAddress uidAddr = {
            kAudioDevicePropertyDeviceUID,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        CFStringRef uidRef = NULL;
        UInt32 uidSize = sizeof(uidRef);
        status = AudioObjectGetPropertyData(deviceIDs[i], &uidAddr, 0, NULL, &uidSize, &uidRef);
        if (status == noErr && uidRef != NULL) {
            cfstring_to_cstr(uidRef, devices[i].uid, sizeof(devices[i].uid));
            CFRelease(uidRef);
        }

        // Transport type
        AudioObjectPropertyAddress transportAddr = {
            kAudioDevicePropertyTransportType,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        UInt32 transportType = 0;
        UInt32 transportSize = sizeof(transportType);
        status = AudioObjectGetPropertyData(deviceIDs[i], &transportAddr, 0, NULL, &transportSize, &transportType);
        if (status == noErr) {
            transport_type_str(transportType, devices[i].transportType, sizeof(devices[i].transportType));
        } else {
            strncpy(devices[i].transportType, "unknown", sizeof(devices[i].transportType));
        }

        // Input/output capability
        devices[i].hasInput = device_has_streams(deviceIDs[i], kAudioObjectPropertyScopeInput);
        devices[i].hasOutput = device_has_streams(deviceIDs[i], kAudioObjectPropertyScopeOutput);
    }

    *outCount = count;
    return 0;
}

static int get_default_device_uid(AudioObjectPropertySelector selector, char *outUID, int maxLen) {
    AudioObjectPropertyAddress addr = {
        selector,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };

    AudioObjectID deviceID = kAudioObjectUnknown;
    UInt32 size = sizeof(deviceID);
    OSStatus status = AudioObjectGetPropertyData(
        kAudioObjectSystemObject, &addr, 0, NULL, &size, &deviceID);
    if (status != noErr) return status;

    AudioObjectPropertyAddress uidAddr = {
        kAudioDevicePropertyDeviceUID,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    CFStringRef uidRef = NULL;
    UInt32 uidSize = sizeof(uidRef);
    status = AudioObjectGetPropertyData(deviceID, &uidAddr, 0, NULL, &uidSize, &uidRef);
    if (status != noErr) return status;
    if (uidRef == NULL) return -1;

    cfstring_to_cstr(uidRef, outUID, maxLen);
    CFRelease(uidRef);
    return 0;
}

int mux_get_default_output_uid(char *outUID, int maxLen) {
    return get_default_device_uid(kAudioHardwarePropertyDefaultOutputDevice, outUID, maxLen);
}

int mux_get_default_input_uid(char *outUID, int maxLen) {
    return get_default_device_uid(kAudioHardwarePropertyDefaultInputDevice, outUID, maxLen);
}

static int set_default_device(AudioObjectPropertySelector selector, const char *uid) {
    AudioObjectID deviceID = find_device_by_uid(uid);
    if (deviceID == kAudioObjectUnknown) return -1;

    AudioObjectPropertyAddress addr = {
        selector,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    return AudioObjectSetPropertyData(
        kAudioObjectSystemObject, &addr, 0, NULL, sizeof(deviceID), &deviceID);
}

int mux_set_default_output_device(const char *uid) {
    return set_default_device(kAudioHardwarePropertyDefaultOutputDevice, uid);
}

int mux_set_default_input_device(const char *uid) {
    return set_default_device(kAudioHardwarePropertyDefaultInputDevice, uid);
}

// Volume strategy: try HAL kAudioDevicePropertyVolumeScalar first (master, then per-channel),
// fall back to AudioHardwareService VirtualMainVolume for devices that only expose volume
// through that API (e.g. some USB audio interfaces).

typedef enum {
    VOL_METHOD_NONE = 0,
    VOL_METHOD_HAL_MASTER,
    VOL_METHOD_HAL_CHANNELS,
    VOL_METHOD_SERVICE,
} VolumeMethod;

static VolumeMethod detect_volume_method(AudioObjectID deviceID, AudioObjectPropertyScope propScope, int *outChannelCount) {
    *outChannelCount = 0;

    // 1. HAL master channel (element 0)
    AudioObjectPropertyAddress addr = {
        kAudioDevicePropertyVolumeScalar,
        propScope,
        kAudioObjectPropertyElementMain
    };
    if (AudioObjectHasProperty(deviceID, &addr)) {
        return VOL_METHOD_HAL_MASTER;
    }

    // 2. HAL per-channel
    AudioObjectPropertyAddress streamAddr = {
        kAudioDevicePropertyStreamConfiguration,
        propScope,
        kAudioObjectPropertyElementMain
    };
    UInt32 size = 0;
    OSStatus status = AudioObjectGetPropertyDataSize(deviceID, &streamAddr, 0, NULL, &size);
    if (status == noErr && size > 0) {
        AudioBufferList *bufList = (AudioBufferList *)malloc(size);
        status = AudioObjectGetPropertyData(deviceID, &streamAddr, 0, NULL, &size, bufList);
        if (status == noErr) {
            int channels = 0;
            for (UInt32 i = 0; i < bufList->mNumberBuffers; i++) {
                channels += bufList->mBuffers[i].mNumberChannels;
            }
            free(bufList);

            addr.mElement = 1;
            if (channels > 0 && AudioObjectHasProperty(deviceID, &addr)) {
                *outChannelCount = channels;
                return VOL_METHOD_HAL_CHANNELS;
            }
        } else {
            free(bufList);
        }
    }

    // 3. AudioHardwareService VirtualMainVolume
    AudioObjectPropertyAddress svcAddr = {
        kAudioHardwareServiceDeviceProperty_VirtualMainVolume,
        propScope,
        kAudioObjectPropertyElementMain
    };
    if (AudioObjectHasProperty(deviceID, &svcAddr)) {
        return VOL_METHOD_SERVICE;
    }

    return VOL_METHOD_NONE;
}

int mux_get_device_volume(const char *uid, int scope, float *outVolume, int *hasVolume) {
    AudioObjectID deviceID = find_device_by_uid(uid);
    if (deviceID == kAudioObjectUnknown) return -1;

    AudioObjectPropertyScope propScope =
        (scope == 0) ? kAudioObjectPropertyScopeOutput : kAudioObjectPropertyScopeInput;

    int channelCount = 0;
    VolumeMethod method = detect_volume_method(deviceID, propScope, &channelCount);

    if (method == VOL_METHOD_NONE) {
        *hasVolume = 0;
        *outVolume = 0;
        return 0;
    }

    *hasVolume = 1;

    if (method == VOL_METHOD_SERVICE) {
        AudioObjectPropertyAddress addr = {
            kAudioHardwareServiceDeviceProperty_VirtualMainVolume,
            propScope,
            kAudioObjectPropertyElementMain
        };
        Float32 volume = 0;
        UInt32 size = sizeof(volume);
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
        OSStatus status = AudioHardwareServiceGetPropertyData(deviceID, &addr, 0, NULL, &size, &volume);
#pragma clang diagnostic pop
        if (status != noErr) return status;
        *outVolume = volume;
        return 0;
    }

    AudioObjectPropertyAddress addr = {
        kAudioDevicePropertyVolumeScalar,
        propScope,
        kAudioObjectPropertyElementMain
    };

    if (method == VOL_METHOD_HAL_MASTER) {
        Float32 volume = 0;
        UInt32 size = sizeof(volume);
        OSStatus status = AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &volume);
        if (status != noErr) return status;
        *outVolume = volume;
    } else {
        Float32 total = 0;
        int read = 0;
        for (int ch = 1; ch <= channelCount; ch++) {
            addr.mElement = ch;
            Float32 volume = 0;
            UInt32 size = sizeof(volume);
            OSStatus status = AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &volume);
            if (status == noErr) {
                total += volume;
                read++;
            }
        }
        if (read > 0) {
            *outVolume = total / read;
        } else {
            *hasVolume = 0;
            *outVolume = 0;
        }
    }

    return 0;
}

int mux_set_device_volume(const char *uid, int scope, float volume) {
    AudioObjectID deviceID = find_device_by_uid(uid);
    if (deviceID == kAudioObjectUnknown) return -1;

    AudioObjectPropertyScope propScope =
        (scope == 0) ? kAudioObjectPropertyScopeOutput : kAudioObjectPropertyScopeInput;

    int channelCount = 0;
    VolumeMethod method = detect_volume_method(deviceID, propScope, &channelCount);

    if (method == VOL_METHOD_NONE) return 0;

    Float32 vol = volume;

    if (method == VOL_METHOD_SERVICE) {
        AudioObjectPropertyAddress addr = {
            kAudioHardwareServiceDeviceProperty_VirtualMainVolume,
            propScope,
            kAudioObjectPropertyElementMain
        };
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
        return AudioHardwareServiceSetPropertyData(deviceID, &addr, 0, NULL, sizeof(vol), &vol);
#pragma clang diagnostic pop
    }

    AudioObjectPropertyAddress addr = {
        kAudioDevicePropertyVolumeScalar,
        propScope,
        kAudioObjectPropertyElementMain
    };

    if (method == VOL_METHOD_HAL_MASTER) {
        return AudioObjectSetPropertyData(deviceID, &addr, 0, NULL, sizeof(vol), &vol);
    }

    OSStatus lastErr = noErr;
    for (int ch = 1; ch <= channelCount; ch++) {
        addr.mElement = ch;
        if (AudioObjectHasProperty(deviceID, &addr)) {
            OSStatus status = AudioObjectSetPropertyData(deviceID, &addr, 0, NULL, sizeof(vol), &vol);
            if (status != noErr) lastErr = status;
        }
    }
    return lastErr;
}

int mux_get_volume_method(const char *uid, int scope) {
    AudioObjectID deviceID = find_device_by_uid(uid);
    if (deviceID == kAudioObjectUnknown) return -1;
    AudioObjectPropertyScope propScope =
        (scope == 0) ? kAudioObjectPropertyScopeOutput : kAudioObjectPropertyScopeInput;
    int channelCount = 0;
    return (int)detect_volume_method(deviceID, propScope, &channelCount);
}
