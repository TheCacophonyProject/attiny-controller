// Copyright 2017 The Cacophony Project. All rights reserved.
// Use of this source code is governed by the Apache License Version 2.0;
// see the LICENSE file for further details.

package main

import (
	"errors"
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type AttinyConfig struct {
	PiWakeUpTime time.Time
	PiSleepTime  time.Time
}

func (conf *AttinyConfig) Validate() error {
	if conf.PiSleepTime.IsZero() && !conf.PiWakeUpTime.IsZero() {
		return errors.New("pi-sleep-time is set but pi-wake-up-time isn't")
	}
	if !conf.PiSleepTime.IsZero() && conf.PiWakeUpTime.IsZero() {
		return errors.New("pi-wake-up-time is set but pi-sleep-time isn't")
	}
	return nil
}

type rawConfig struct {
	PiWakeUp string `yaml:"pi-wake-up-time"`
	PiSleep  string `yaml:"pi-sleep-time"`
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

	conf := &AttinyConfig{}

	const timeOnly = "15:04"
	if raw.PiWakeUp != "" {
		t, err := time.Parse(timeOnly, raw.PiWakeUp)
		if err != nil {
			return nil, errors.New("invalid Pi wake up time")
		}
		conf.PiWakeUpTime = t
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
