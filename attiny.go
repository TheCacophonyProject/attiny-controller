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
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/host"
)

const (
	wantedVersion = 4

	attinyAddress = 0x04

	watchdogReg         = 0x12
	sleepReg            = 0x11
	batteryVoltageLoReg = 0x20
	batteryVoltageHiReg = 0x21
	wifiStateReg        = 0x13
	versionReg          = 0x22

	// 3 was just a randomly chosen as the number for the attiny to return
	// to indicate its presence.
	magicReturn = 0x03

	// Check for the ATtiny for up to a minute.
	maxConnectAttempts     = 20
	connectAttemptInterval = 3 * time.Second

	// Parameters for transaction retries.
	maxTxAttempts   = 5
	txRetryInterval = time.Second

	wifiInterface = "wlan0" // If this is changed also change it in /_release/10-notify-attiny to match
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

	a := &attiny{dev: dev, voltages: voltages}
	if err := a.getVersion(); err != nil {
		return nil, err
	}
	log.Printf("attiny version: %d\n", a.version)
	if wantedVersion > a.version {
		log.Printf("wanted attiny version %d or higher. Have version %d."+
			" Some features won't be available\n", wantedVersion, a.version)
	}
	return a, nil
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
	mu      sync.Mutex
	dev     *i2c.Dev
	version uint8

	voltages         Voltages
	checkedOnBattery bool
	onBattery        bool

	wifiMu             sync.Mutex
	wifiConnectedState bool
}

func (a *attiny) getVersion() error {
	version, err := a.readUint8(versionReg)
	if err != nil {
		return err
	}
	a.version = version
	return nil
}

func (a *attiny) readUint8(reg byte) (uint8, error) {
	i := make([]byte, 1)
	if err := a.tx(i, []byte{reg}); err != nil {
		return 0, err
	}
	return uint8(i[0]), nil
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

func (a *attiny) versionCheck(requiredVersion uint8) error {
	fpcs := make([]uintptr, 1)
	runtime.Callers(2, fpcs)
	caller := runtime.FuncForPC(fpcs[0] - 1)
	if a.version < requiredVersion {
		return fmt.Errorf("attiny version was %d and needs version %d or above for '%s'", a.version, requiredVersion, caller.Name())
	}
	return nil
}

func (a *attiny) UpdateWifiState() error {
	if err := a.versionCheck(4); err != nil {
		return err
	}

	a.wifiMu.Lock()
	defer a.wifiMu.Unlock()
	outByte, err := exec.Command("ip", "a", "show", wifiInterface).Output()
	if err != nil {
		return err
	}
	newState := strings.Contains(string(outByte), "state UP")
	if a.wifiConnectedState == newState {
		return nil
	}
	var b byte = 0x00
	if newState {
		b = 0x01
	}
	err = a.write(wifiStateReg, []byte{b})
	if err == nil {
		a.wifiConnectedState = newState
		log.Printf("updated wifi connected state to '%t'", a.wifiConnectedState)
	}
	return err
}

func (a *attiny) checkIsOnBattery() (bool, error) {
	if err := a.versionCheck(4); err != nil {
		return false, err
	}
	if a.checkedOnBattery {
		return a.onBattery, nil
	}
	batVal, err := a.readBatteryValue()
	if err != nil {
		return false, err
	}
	a.onBattery = batVal > a.voltages.NoBattery
	a.checkedOnBattery = true
	return a.onBattery, nil
}

// readBatteryValue will get the analog value read by the attiny on the battery sense pin
func (a *attiny) readBatteryValue() (uint16, error) {
	if err := a.versionCheck(4); err != nil {
		return 0, err
	}
	if !a.voltages.Enable {
		return 0, nil
	}
	l := make([]byte, 1)
	h := make([]byte, 1)
	if err := a.tx(l, []byte{batteryVoltageLoReg}); err != nil {
		return 0, err
	}
	if err := a.tx(h, []byte{batteryVoltageHiReg}); err != nil {
		return 0, err
	}
	if h[0] == 127 || l[0] == 127 {
		return a.readBatteryValue()
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
