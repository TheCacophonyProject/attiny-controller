/*
attiny-controller - Communicates with ATtiny microcontroller
Copyright (C) 2018, The Cacophony Project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"io/ioutil"
	"os"

	"github.com/TheCacophonyProject/attiny-controller/location"
	"github.com/TheCacophonyProject/window"
	yaml "gopkg.in/yaml.v2"
)

type AttinyConfig struct {
	OnWindow *window.Window
	Voltages Voltages
}

type rawConfig struct {
	PiWakeUp string   `yaml:"pi-wake-time"`
	PiSleep  string   `yaml:"pi-sleep-time"`
	Voltages Voltages `yaml:"voltages"`
}

type Voltages struct {
	Enable      bool   `yaml:"enable"`       // Enable reading voltage through ATtiny
	NoBattery   uint16 `yaml:"no-battery"`   // If voltage reading is less than this it is not powered by a battery
	LowBattery  uint16 `yaml:"low-battery"`  // Voltage of a low battery
	FullBattery uint16 `yaml:"full-battery"` // Voltage of a full battery
}

func ParseAttinyConfigFile(filename, locationFile string) (*AttinyConfig, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	locationBuf, err := ioutil.ReadFile(locationFile)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	loc, err := location.New(locationBuf)
	if err != nil {
		return nil, err
	}

	return ParseAttinyConfig(buf, loc)
}

func ParseAttinyConfig(buf []byte, loc *location.LocationConfig) (*AttinyConfig, error) {
	raw := rawConfig{}
	if err := yaml.Unmarshal(buf, &raw); err != nil {
		return nil, err
	}

	conf := &AttinyConfig{
		Voltages: raw.Voltages,
	}

	w, err := window.New(raw.PiWakeUp, raw.PiSleep, loc.Latitude, loc.Longitude)
	if err != nil {
		return nil, err
	}
	conf.OnWindow = w

	return conf, nil
}
