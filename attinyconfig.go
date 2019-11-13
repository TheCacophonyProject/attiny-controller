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
	Battery  config.Battery
}

func ParseConfig(configDir string) (*AttinyConfig, error) {
	rawConfig, err := config.New(configDir)
	if err != nil {
		return nil, err
	}

	windows := config.DefaultWindows()
	if err := rawConfig.Unmarshal(config.WindowsKey, &windows); err != nil {
		return nil, err
	}

	location := config.DefaultWindowLocation()
	rawConfig.Unmarshal(config.LocationKey, &location)

	battery := config.DefaultBattery()
	if err := rawConfig.Unmarshal(config.BatteryKey, &battery); err != nil {
		return nil, err
	}

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
		Battery:  battery,
	}, nil
}
