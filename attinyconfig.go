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
	"github.com/TheCacophonyProject/go-config"
	"github.com/TheCacophonyProject/window"
)

type AttinyConfig struct {
	OnWindow *window.Window
	Voltages Voltages
}

type Voltages struct {
	Enable      bool   // Enable reading voltage through ATtiny
	NoBattery   uint16 // If voltage reading is less than this it is not powered by a battery
	LowBattery  uint16 // Voltage of a low battery
	FullBattery uint16 // Voltage of a full battery
}

func ParseConfig(configDir string) (*AttinyConfig, error) {
	rawConfig, err := config.New(configDir)
	if err != nil {
		return nil, err
	}

	var battery config.Battery
	rawConfig.Unmarshal("battery", &battery)

	windows := config.DefaultWindows
	rawConfig.Unmarshal(config.WindowsKey, windows)

	location := config.DefaultLocation
	rawConfig.Unmarshal(config.LocationKey, location)

	w, err := window.New(
		windows.PowerOn,
		windows.PowerOff,
		float64(location.Latitude),
		float64(location.Longitude))
	if err != nil {
		return nil, err
	}

	return &AttinyConfig{
		OnWindow: w,
		Voltages: Voltages{
			Enable:      battery.EnableVoltageReadings,
			NoBattery:   battery.NoBattery,
			LowBattery:  battery.LowBattery,
			FullBattery: battery.FullBattery,
		},
	}, nil
}
