package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
)

// Errors that are returns for unknown device types
var (
	ErrUnknownDeviceType    = errors.New("Unknown device type")
	ErrUnknownDeviceSubType = errors.New("Unknown device sub type")
)

// Config represents the overall config of the application
type Config struct {
	API     APIConfig `json:"api"`
	Devices []Device  `json:"devices"`
}

// ReadConfig reads the config from the configuration JSON file
func ReadConfig() (Config, error) {
	config := new(Config)
	contents, err := ioutil.ReadFile("config.json")
	if err != nil {
		return *config, err
	}

	// Unmarshal all the common properties first
	err = json.Unmarshal(contents, config)
	if err != nil {
		return *config, err
	}

	// Also unmarshal the device json in to RawMessages to work out the what device info types to unmarshal
	var rawConfig map[string]*json.RawMessage
	err = json.Unmarshal(contents, &rawConfig)
	if err != nil {
		return *config, err
	}

	devicesArray := rawConfig["devices"]
	var rawDevices []*json.RawMessage
	err = json.Unmarshal(*devicesArray, &rawDevices)
	if err != nil {
		return *config, err
	}

	// For each device, unmarshal the device info based on its type
	for i, v := range config.Devices {
		info, err := unmarshalInfo(*rawDevices[i], v.Type, v.SubType)
		if err != nil {
			return *config, err
		}

		v.Info = info
	}

	return *config, nil
}

func unmarshalInfo(raw []byte, deviceType string, deviceSubType string) (interface{}, error) {
	switch deviceType {
	case TypeTv:
		return unmarshalTvInfo(raw, deviceSubType)
	default:
		return nil, ErrUnknownDeviceType
	}
}

func unmarshalTvInfo(raw []byte, tvType string) (interface{}, error) {
	var info interface{}
	switch tvType {
	case SubTypeWebOsTv:
		info = new(WebOsTvDeviceInfo)
	default:
		return nil, ErrUnknownDeviceSubType
	}

	err := json.Unmarshal(raw, info)
	if err != nil {
		return nil, err
	}

	return info, nil
}
