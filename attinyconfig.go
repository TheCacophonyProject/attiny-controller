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
	"errors"
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type AttinyConfig struct {
	PiWakeTime  time.Time
	PiSleepTime time.Time
	Voltages    Voltages
}

func (conf *AttinyConfig) Validate() error {
	if conf.PiSleepTime.IsZero() && !conf.PiWakeTime.IsZero() {
		return errors.New("pi-sleep-time is set but pi-wake-time isn't")
	}
	if !conf.PiSleepTime.IsZero() && conf.PiWakeTime.IsZero() {
		return errors.New("pi-wake-time is set but pi-sleep-time isn't")
	}
	return nil
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

func ParseAttinyConfigFile(filename string) (*AttinyConfig, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return ParseAttinyConfig(buf)
}

func ParseAttinyConfig(buf []byte) (*AttinyConfig, error) {
	raw := rawConfig{}
	if err := yaml.Unmarshal(buf, &raw); err != nil {
		return nil, err
	}

	conf := &AttinyConfig{
		Voltages: raw.Voltages,
	}

	const timeOnly = "15:04"
	if raw.PiWakeUp != "" {
		t, err := time.Parse(timeOnly, raw.PiWakeUp)
		if err != nil {
			return nil, errors.New("invalid Pi wake up time")
		}
		conf.PiWakeTime = t
	}

	if raw.PiSleep != "" {
		t, err := time.Parse(timeOnly, raw.PiSleep)
		if err != nil {
			return nil, errors.New("invalid Pi sleep time")
		}
		conf.PiSleepTime = t
	}

	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return conf, nil
}
