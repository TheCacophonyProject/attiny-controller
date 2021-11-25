package main

import (
	"log"
	"time"

	api "github.com/TheCacophonyProject/go-api"
	"github.com/TheCacophonyProject/window"
)

const interval = 4 * time.Hour
const maxAttempts = 3
const attemptDelay = 5 * time.Second

const heartBeatDelay = 30 * time.Minute

type Heartbeat struct {
	api         *api.CacophonyAPI
	window      *window.Window
	validUntil  time.Time
	end         time.Time
	penultimate bool
}

// Used to test
type Clock interface {
	Sleep(d time.Duration)
	Now() time.Time
}

type HeartBeatClock struct {
}

func (h *HeartBeatClock) Sleep(d time.Duration) {
	time.Sleep(d)
}
func (h *HeartBeatClock) Now() time.Time {
	return time.Now()
}

var clock Clock = &HeartBeatClock{}

func heartBeatLoop(window *window.Window) {
	hb, err := NewHeartbeat(window)
	if err != nil {
		log.Printf("Error starting up heart beat %v", err)
		return
	}
	sendBeats(hb, window)
}
func sendBeats(hb *Heartbeat, window *window.Window) {
	initialDelay := heartBeatDelay

	if !window.Active() {
		until := window.Until()
		if until > initialDelay {
			initialDelay = until
		}
	}
	log.Printf("Sending initial heart beat in %v", initialDelay)
	clock.Sleep(initialDelay)
	for {
		attempt := 0
		done := hb.updateNextBeat()

		for attempt < maxAttempts {
			err := sendHeartbeat(hb.api, hb.validUntil)
			if err == nil {
				break
			}
			log.Printf("Error sending heart beat sleeping and trying again: %v", err)
			clock.Sleep(attemptDelay)
		}
		if done {
			log.Printf("Sent penultimate heartbeat")
			return
		}

		nextEventIn := hb.validUntil.Sub(clock.Now())
		if !hb.penultimate && nextEventIn >= 2*time.Hour {
			nextEventIn = nextEventIn - 1*time.Hour
		} else {
			// 5 minutes to give a bit of leeway
			nextEventIn = nextEventIn - 5*time.Minute

		}
		log.Printf("Heartbeat sleeping until %v", clock.Now().Add(nextEventIn))
		clock.Sleep(nextEventIn)
	}
}

func NewHeartbeat(window *window.Window) (*Heartbeat, error) {
	var nextEnd time.Time
	if !window.NoWindow {
		nextEnd = window.NextEnd()
	}

	apiClient, err := api.New()
	if err != nil {
		log.Printf("Error connecting to api %v", apiClient)
	}

	h := &Heartbeat{api: apiClient, end: nextEnd, window: window}
	return h, err
}

//updates next heart beat time, returns true if will be the final event
func (h *Heartbeat) updateNextBeat() bool {
	if h.penultimate {
		h.validUntil = h.end
		return true
	}
	h.validUntil = clock.Now().Add(interval)
	if !h.window.NoWindow && h.validUntil.After(h.end.Add(-time.Hour)) {
		// always want an event 1 hour before end if possible
		h.validUntil = h.end.Add(-time.Hour)
		if clock.Now().After(h.validUntil) {
			// rare case of very short window
			h.validUntil = h.end
			return true
		}
		h.penultimate = true
	}
	return false
}

func sendHeartbeat(api *api.CacophonyAPI, nextBeat time.Time) error {
	if api == nil {
		return nil
	}
	_, err := api.Heartbeat(nextBeat)
	if err == nil {
	}
	log.Printf("Sent heart, valid until %v", nextBeat)
	return err
}

func sendFinalHeartBeat(window *window.Window) error {
	apiClient, err := api.New()
	if err != nil {
		log.Printf("Error connecting to api %v", apiClient)
		return err
	}
	return sendHeartbeat(apiClient, window.NextStart().Add(heartBeatDelay*2))
}
