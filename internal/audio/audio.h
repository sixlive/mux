#ifndef MUX_AUDIO_H
#define MUX_AUDIO_H

#include <CoreAudio/CoreAudio.h>

typedef struct {
    AudioObjectID deviceID;
    char name[256];
    char uid[256];
    char transportType[64];
    int hasInput;
    int hasOutput;
} MuxAudioDevice;

int mux_get_devices(MuxAudioDevice *devices, int maxDevices, int *outCount);
int mux_set_default_output_device(const char *uid);
int mux_set_default_input_device(const char *uid);
int mux_get_default_output_uid(char *outUID, int maxLen);
int mux_get_default_input_uid(char *outUID, int maxLen);
int mux_get_device_volume(const char *uid, int scope, float *outVolume, int *hasVolume);
int mux_set_device_volume(const char *uid, int scope, float volume);
int mux_get_volume_method(const char *uid, int scope);

#endif
