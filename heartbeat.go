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
	api       *api.CacophonyAPI
	window    *window.Window
	nextEvent time.Time
	end       time.Time
}

func heartBeatLoop(window *window.Window) {
	if !window.Active() {
		log.Printf("After active window not sending heart beat")
		return
	}

	hb, err := NewHeartbeat(window)
	if err != nil {
		log.Printf("Error starting up heart beat %v", err)
		return
	}
	log.Printf("Sending initial heart beat in %v", heartBeatDelay)
	time.Sleep(heartBeatDelay)

	for {
		attempt := 0
		done := hb.updateNextBeat()
		for attempt < maxAttempts {
			err := sendHeartbeat(hb.api, hb.nextEvent)
			if err == nil {
				break
			}
			log.Printf("Error sending heart beat sleeping and trying again: %v", err)
			time.Sleep(attemptDelay)
		}
		if done {
			log.Printf("Sent final heartbeat")
			return
		}

		nextEventIn := hb.nextEvent.Sub(time.Now())
		if nextEventIn >= 2*time.Hour {
			nextEventIn = nextEventIn - 1*time.Hour
		} else {
			// 5 minutes to give a bit of leeway
			nextEventIn = nextEventIn - 5*time.Minute

		}
		log.Printf("Heartbeat sleeping until %v", time.Now().Add(nextEventIn))
		time.Sleep(nextEventIn)
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
		return nil, err
	}

	h := &Heartbeat{api: apiClient, end: nextEnd, window: window}
	return h, nil
}

//updates next heart beat time, returns true if this was the final event
func (h *Heartbeat) updateNextBeat() bool {
	h.nextEvent = time.Now().Add(interval)
	if !h.window.NoWindow && h.nextEvent.After(h.end.Add(-time.Hour)) {
		h.nextEvent = h.end
		return true
	}
	return false
}

func sendHeartbeat(api *api.CacophonyAPI, nextBeat time.Time) error {
	_, err := api.Heartbeat(nextBeat)
	if err == nil {
		log.Printf("Sent heart, valid until %v", nextBeat)
	}
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
