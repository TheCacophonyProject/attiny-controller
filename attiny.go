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
	"encoding/binary"
	"sync"
	"time"

	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/conn/mmr"
	"periph.io/x/periph/host"
)

const (
	attinyAddress = 0x04

	// 3 was just a randomly chosen as the number for the attiny to return
	// to indicate its presence.
	magicReturn = 0x03

	// Check for the ATtiny for up to a minute.
	connectAttempts        = 20
	connectAttemptInterval = 3 * time.Second

	watchdogTimerAddress = 0x12
	sleepAddress         = 0x11
)

// connectATtiny sets up a i2c device for talking to the ATtiny and
// returns a wrapper for it. If no ATtiny was detected (nil, nil) will
// be returned.
func connectATtiny() (*attiny, error) {
	if _, err := host.Init(); err != nil {
		return nil, err
	}

	bus, err := i2creg.Open("")
	if err != nil {
		return nil, err
	}

	dev := &mmr.Dev8{
		Conn:  &i2c.Dev{Bus: bus, Addr: attinyAddress},
		Order: binary.LittleEndian,
	}

	if !detectATtiny(dev) {
		bus.Close()
		return nil, nil
	}
	return &attiny{dev: dev}, nil
}

func detectATtiny(dev *mmr.Dev8) bool {
	for i := 0; i < connectAttempts; i++ {
		time.Sleep(connectAttemptInterval)

		value, err := dev.ReadUint8(0)
		if err == nil && value == magicReturn {
			return true
		}
	}
	return false
}

type attiny struct {
	mu  sync.Mutex
	dev *mmr.Dev8
}

// PowerOff asks the ATtiny to turn the system off for the number of
// minutes specified.
func (a *attiny) PowerOff(minutes int) error {
	if minutes <= 0 {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.dev.WriteUint16(sleepAddress, uint16(minutes))
}

// PingWatchdog ping's the ATTiny's watchdog timer to prevent it from
// rebooting the system.
func (a *attiny) PingWatchdog() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.dev.Conn.Tx([]byte{watchdogTimerAddress}, nil)
}
