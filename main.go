package main

import (
	"log"
	"time"

	"github.com/TheCacophonyProject/window"
	arg "github.com/alexflint/go-arg"
	i2c "golang.org/x/exp/io/i2c"
)

const wdtAddress = 0x12
const sleepAddress = 0x11

var version = "No version provided"

type Args struct {
	ConfigFile string `arg:"-c,--config" help:"path to configuration file"`
	SkipWait   bool   `arg:"-s,--skip-wait" help:"will not wait for the date to update"`
}

func (Args) Version() string {
	return version
}

func procArgs() Args {
	var args Args
	args.ConfigFile = "/etc/thermal-recorder.yaml"
	arg.MustParse(&args)
	return args
}

func main() {
	err := runMain()
	if err != nil {
		log.Fatal(err)
	}
}

func runMain() error {
	log.SetFlags(0) // Removes default timestamp flag

	args := procArgs()
	log.Printf("running version: %s", version)

	attiny, err := i2c.Open(&i2c.Devfs{Dev: "/dev/i2c-1"}, 0x04)
	if err != nil {
		return err
	}
	defer attiny.Close()

	buf := make([]byte, 1)
	attiny.Read(buf)
	if buf[0] != 3 { // 3 was just a raddomly chosen number for the attiny to return.
		log.Println("attiny not connected")
		return nil
	}
	log.Println("connected to attiny")
	go restartWDT(attiny)
	if !args.SkipWait {
		time.Sleep(2 * time.Minute) // Wait for date to be updated.
	}

	for {
		conf, err := ParseConfigFile(args.ConfigFile)
		if err != nil {
			return err
		}
		window := window.New(conf.WindowStart, conf.WindowEnd)
		minutesUntilActive := int(window.Until().Minutes())
		log.Printf("minutes until active %d", minutesUntilActive)
		if minutesUntilActive > 15 {
			minutesUntilActive = minutesUntilActive - 10
			lb := byte(minutesUntilActive / 256)
			rb := byte(minutesUntilActive % 256)
			err = attiny.Write([]byte{sleepAddress, lb, rb})
			if err != nil {
				log.Fatal(err)
			}
		}
		time.Sleep(time.Minute * 5)
	}
	return nil
}

func restartWDT(attiny *i2c.Device) {
	log.Println("startign WDT signals to attiny")
	for {
		err := attiny.Write([]byte{wdtAddress})
		if err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Minute)
	}
}
