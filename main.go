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
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	linuxproc "github.com/c9s/goprocinfo/linux"

	"github.com/TheCacophonyProject/go-config"
	arg "github.com/alexflint/go-arg"
	"golang.org/x/sys/unix"
)

// How long to wait before checking the recording window. This
// gives time to do something with the device before it turns off.
const (
	initialGracePeriod     = 20 * time.Minute
	batteryCSVFile         = "/var/log/battery.csv"
	batteryReadingInterval = 10 * time.Minute
	systemStatFile         = "/proc/stat"
)

var (
	version = "<not set>"

	mu          sync.Mutex
	stayOnUntil = time.Now()
)

func shouldTurnOff(minutesUntilActive int) bool {
	mu.Lock()
	defer mu.Unlock()
	if time.Now().Before(stayOnUntil) {
		return false
	}
	return minutesUntilActive > 15
}

func setStayOnUntil(newTime time.Time) error {
	if time.Until(newTime) > 12*time.Hour {
		return errors.New("can not delay over 12 hours")
	}
	mu.Lock()
	stayOnUntil = newTime
	mu.Unlock()
	log.Println("staying on until", newTime.Format(time.UnixDate))
	return nil
}

type Args struct {
	ConfigDir          string `arg:"-c,--config" help:"configuration folder"`
	SkipWait           bool   `arg:"-s,--skip-wait" help:"will not wait for the date to update"`
	Timestamps         bool   `arg:"-t,--timestamps" help:"include timestamps in log output"`
	SkipSystemShutdown bool   `arg:"--skip-system-shutdown" help:"don't shut down operating system when powering down"`
}

func (Args) Version() string {
	return version
}

func procArgs() Args {
	args := Args{
		ConfigDir: config.DefaultConfigDir,
	}
	arg.MustParse(&args)
	return args
}

func main() {
	err := runMain()
	if err != nil {
		log.Fatal(err)
	}
	// If no error then keep the background goroutines running.
	runtime.Goexit()
}

func runMain() error {
	args := procArgs()

	if !args.Timestamps {
		log.SetFlags(0)
	}

	log.Printf("running version: %s", version)

	conf, err := ParseConfig(args.ConfigDir)
	if err != nil {
		log.Printf("error parsing config: %s\nwill try to just ping watchdog", err)
		return justPingWatchdog()
	}

	log.Println("connecting to attiny")
	attiny, err := connectATtiny(conf.Battery)
	if err != nil {
		return err
	}
	if attiny == nil {
		log.Println("attiny not present")
		return nil
	}
	log.Println("connected to attiny")

	if onBattery, err := attiny.checkIsOnBattery(); err != nil {
		log.Println(err.Error())
	} else if onBattery {
		log.Println("on battery power")
	} else {
		log.Println("not on battery")
	}

	log.Println("starting D-Bus service")
	if err := startService(attiny); err != nil {
		return err
	}
	log.Println("started D-Bus service")

	go updateWatchdogTimer(attiny)
	if err := attiny.UpdateWifiState(); err != nil {
		log.Println("failed to update wifi state:", err)
	}

	go batteryLoop(attiny)

	log.Printf("on window: %s", conf.OnWindow)

	if conf.OnWindow.NoWindow {
		log.Printf("no window active so pinging watchdog only")
		runtime.Goexit()
	}

	if !args.SkipWait {
		log.Printf("waiting for %s before applying recording window", initialGracePeriod)
		time.Sleep(initialGracePeriod)
	}

	for {
		if conf.OnWindow.Active() {
			untilEnd := conf.OnWindow.UntilEnd()
			log.Printf("%s until on window ends", untilEnd)
			log.Println("sleeping until end of window")
			time.Sleep(untilEnd)
		} else {
			minutesUntilActive := int(conf.OnWindow.Until().Minutes())
			log.Printf("minutes until active %d", minutesUntilActive)
			if shouldTurnOff(minutesUntilActive) {
				log.Println("syncing filesystems...")
				unix.Sync()

				log.Println("requesting power off...")
				if err := attiny.PowerOff(minutesUntilActive - 2); err != nil {
					log.Fatal(err)
				}
				log.Println("power off requested")

				if !args.SkipSystemShutdown {
					log.Println("shutting down system...")
					if err := shutdown(); err != nil {
						log.Fatal(err)
					}
				}
			}
			time.Sleep(time.Minute * 5)
		}
	}
}

func updateWatchdogTimer(a *attiny) {
	log.Println("sending watchdog timer updates")
	for {
		if err := a.PingWatchdog(); err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Minute)
	}
}

func batteryLoop(a *attiny) {
	for {
		cpu, err := cpuUsage()
		if err != nil {
			log.Printf("error with getting cpu usage: %s", err)
			return
		}
		batteryVal, err := a.readBatteryValue()
		log.Printf("battery reading: %d", batteryVal)
		nowStr := time.Now().Format("2006-01-02 15:04:05")
		dataStr := fmt.Sprintf("%s, %f, %d\n", nowStr, cpu, batteryVal)
		if err := appendToFile(dataStr, batteryCSVFile); err != nil {
			log.Printf("error logging battery value: %s", err)
			return
		}
		time.Sleep(batteryReadingInterval)
	}
}

func cpuUsage() (float64, error) {
	stat1, err := linuxproc.ReadStat(systemStatFile)
	if err != nil {
		return 0, err
	}
	time.Sleep(3 * time.Second)
	stat2, err := linuxproc.ReadStat(systemStatFile)
	if err != nil {
		return 0, err
	}
	if len(stat1.CPUStats) != len(stat1.CPUStats) {
		return 0, errors.New("bad stat file readings")
	}
	var cpuTotal float64
	for i := 0; i < len(stat1.CPUStats); i++ {
		cpu1 := stat1.CPUStats[i]
		cpu2 := stat2.CPUStats[i]

		total1, idle1 := getTotalAndIdleTicks(&cpu1)
		total2, idle2 := getTotalAndIdleTicks(&cpu2)

		totalDiff := total2 - total1
		idleDiff := idle2 - idle1
		cpu := float64(totalDiff-idleDiff) / float64(totalDiff)
		cpuTotal += cpu
	}
	return cpuTotal / float64(len(stat1.CPUStats)), nil
}

func appendToFile(text string, file string) error {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(text)
	return err
}

func getTotalAndIdleTicks(c *linuxproc.CPUStat) (total, idle uint64) {
	idle = c.IOWait + c.Idle
	total = idle + c.User + c.Nice + c.System + c.IRQ + c.SoftIRQ + c.Steal
	return total, idle
}

func shutdown() error {
	cmd := exec.Command("/sbin/poweroff")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("poweroff failed: %v\n%s", err, output)
	}
	return nil
}

func justPingWatchdog() error {
	attiny, err := connectATtiny(config.Battery{})
	if err != nil {
		return err
	}
	if attiny == nil {
		log.Println("attiny not present")
		return nil
	}
	log.Println("connected to attiny")
	go updateWatchdogTimer(attiny)
	return nil
}
