// Copyright 2018 The Cacophony Project. All rights reserved.
// Use of this source code is governed by the Apache License Version 2.0;
// see the LICENSE file for further details.

package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TheCacophonyProject/window"
)

const dateFormat = "15:04"

type TestClock struct {
	now            time.Time
	expectedSleeps []time.Time

	sleepCount int
	t          *testing.T
	hb         *Heartbeat
}

func (h *TestClock) Sleep(d time.Duration) {
	// nextBeat gets updated after sleep skip first
	if h.sleepCount > 0 {
		if h.sleepCount == len(h.expectedSleeps)-1 {
			// penultimate event is only valid for 5 minutes after sleep
			assert.Equal(h.t, h.expectedSleeps[h.sleepCount].Add(5*time.Minute).Format(dateFormat), h.hb.validUntil.Format(dateFormat))

		} else {
			assert.Equal(h.t, h.expectedSleeps[h.sleepCount].Add(1*time.Hour).Format(dateFormat), h.hb.validUntil.Format(dateFormat))
		}
	}
	h.now = h.now.Add(d)
	assert.Equal(h.t, h.now.Format(dateFormat), h.expectedSleeps[h.sleepCount].Format(dateFormat))
	h.sleepCount += 1
}
func (h *TestClock) Now() time.Time {
	return h.now
}
func (h *TestClock) After(d time.Duration) <-chan time.Time {
	// nextBeat gets updated after sleep skip first
	if h.sleepCount > 0 {
		if h.sleepCount == len(h.expectedSleeps)-1 {
			// penultimate event is only valid for 5 minutes after sleep
			assert.Equal(h.t, h.expectedSleeps[h.sleepCount].Add(5*time.Minute).Format(dateFormat), h.hb.validUntil.Format(dateFormat))

		} else {
			assert.Equal(h.t, h.expectedSleeps[h.sleepCount].Add(1*time.Hour).Format(dateFormat), h.hb.validUntil.Format(dateFormat))
		}
	}
	h.now = h.now.Add(d)
	assert.Equal(h.t, h.now.Format(dateFormat), h.expectedSleeps[h.sleepCount].Format(dateFormat))
	h.sleepCount += 1
	s := make(chan time.Time, 10)
	s <- h.now
	return s
}

func TestSmallWindow(t *testing.T) {
	clock := &TestClock{now: time.Now(), t: t}
	w, err := window.New(clock.Now().Format(dateFormat), clock.Now().Add(time.Hour).Format(dateFormat), 0, 0)
	sleeps := make([]time.Time, 1)
	sleeps[0] = clock.now.Add(30 * time.Minute)

	clock.expectedSleeps = sleeps
	require.NoError(t, err)
	heartBeatTestLoop(w, clock)
}
func TestShortDelay(t *testing.T) {
	clock := &TestClock{now: time.Now(), t: t}
	w, err := window.New(clock.Now().Add(10*time.Minute).Format(dateFormat), clock.Now().Add(4*time.Hour).Format(dateFormat), 0, 0)
	sleeps := make([]time.Time, 2, 2)
	sleeps[0] = clock.now.Add(30 * time.Minute)
	sleeps[1] = w.NextEnd().Add(-65 * time.Minute)

	clock.expectedSleeps = sleeps
	require.NoError(t, err)
	heartBeatTestLoop(w, clock)
}

func TestLongDelay(t *testing.T) {
	clock := &TestClock{now: time.Now(), t: t}
	w, err := window.New(clock.Now().Add(time.Hour).Format(dateFormat), clock.Now().Add(4*time.Hour).Format(dateFormat), 0, 0)
	sleeps := make([]time.Time, 2, 2)
	// expect delay until window starts if further than 30 minutes
	sleeps[0] = clock.Now().Add(w.Until())
	sleeps[1] = w.NextEnd().Add(-65 * time.Minute)

	clock.expectedSleeps = sleeps
	require.NoError(t, err)
	heartBeatTestLoop(w, clock)
}

func TestWindow(t *testing.T) {
	clock := &TestClock{now: time.Now(), t: t}
	w, err := window.New(clock.Now().Format(dateFormat), clock.Now().Add(9*time.Hour).Format(dateFormat), 0, 0)
	sleeps := make([]time.Time, 4, 4)
	sleeps[0] = clock.now.Add(30 * time.Minute)
	sleeps[1] = sleeps[0].Add(3 * time.Hour)
	sleeps[2] = sleeps[1].Add(3 * time.Hour)
	sleeps[3] = w.NextEnd().Add(-65 * time.Minute)

	clock.expectedSleeps = sleeps
	require.NoError(t, err)
	heartBeatTestLoop(w, clock)
}

func heartBeatTestLoop(window *window.Window, timer *TestClock) {
	clock = timer
	hb := NewHeartbeat(window)
	hb.MaxAttempts = 1
	timer.hb = hb
	sendBeats(hb, window)
	assert.Equal(timer.t, timer.sleepCount, len(timer.expectedSleeps), "Missing sleep events")
	// assert last beat is at end
	assert.Equal(timer.t, window.NextEnd().Format(dateFormat), hb.validUntil.Format(dateFormat))
}
