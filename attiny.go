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
	"periph.io/x/periph/host"
)

const (
	attinyAddress = 0x04

	watchdogReg = 0x12
	sleepReg    = 0x11

	// 3 was just a randomly chosen as the number for the attiny to return
	// to indicate its presence.
	magicReturn = 0x03

	// Check for the ATtiny for up to a minute.
	maxConnectAttempts     = 20
	connectAttemptInterval = 3 * time.Second

	// Parameters for transaction retries.
	maxTxAttempts   = 5
	txRetryInterval = time.Second
)

// connectATtiny sets up a i2c device for talking to the ATtiny and
// returns a wrapper for it. If no ATtiny was detected (nil, nil) will
// be returned.
func connectATtiny(voltages Voltages) (*attiny, error) {
	if _, err := host.Init(); err != nil {
		return nil, err
	}
	bus, err := i2creg.Open("")
	if err != nil {
		return nil, err
	}
	dev := &i2c.Dev{Bus: bus, Addr: attinyAddress}

	if !detectATtiny(dev) {
		bus.Close()
		return nil, nil
	}
	return &attiny{dev: dev, voltages: voltages}, nil
}

func detectATtiny(dev *i2c.Dev) bool {
	attempts := 0
	for {
		b := make([]byte, 1)
		err := dev.Tx(nil, b)
		if err == nil && b[0] == magicReturn {
			return true
		}

		attempts++
		if attempts >= maxConnectAttempts {
			return false
		}

		time.Sleep(connectAttemptInterval)
	}
}

type attiny struct {
	mu        sync.Mutex
	dev       *i2c.Dev
	onBattery bool
	voltages  Voltages
}

// PowerOff asks the ATtiny to turn the system off for the number of
// minutes specified.
func (a *attiny) PowerOff(minutes int) error {
	if minutes <= 0 {
		return nil
	}
	lb := byte(minutes / 256)
	rb := byte(minutes % 256)
	return a.write(sleepReg, []byte{lb, rb})
}

// PingWatchdog ping's the ATTiny's watchdog timer to prevent it from
// rebooting the system.
func (a *attiny) PingWatchdog() error {
	return a.write(watchdogReg, nil)
}

func (a *attiny) checkIsOnBattery() error {
	batVal, err := a.readBatteryValue()
	if err != nil {
		return err
	}
	a.onBattery = batVal > a.voltages.NoBattery
	return nil
}

// readBatteryValue will get the analog value read by the attiny on the battery sense pin
func (a *attiny) readBatteryValue() (uint16, error) {
	if !a.voltages.Enable {
		return 0, nil
	}
	l := make([]byte, 1)
	h := make([]byte, 1)
	if err := a.tx(l, []byte{0x20}); err != nil {
		return 0, err
	}
	if err := a.tx(h, []byte{0x21}); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16([]byte{h[0], l[0]}), nil
}

func (a *attiny) write(reg uint8, b []byte) error {
	buf := make([]byte, 1, 1+len(b))
	buf[0] = byte(reg)
	buf = append(buf, b...)
	return a.tx(nil, buf)
}

func (a *attiny) tx(w, r []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	attempts := 0
	for {
		err := a.dev.Tx(r, w)
		if err == nil {
			return nil
		}

		attempts++
		if attempts >= maxTxAttempts {
			return err
		}
		time.Sleep(txRetryInterval)
	}
}
