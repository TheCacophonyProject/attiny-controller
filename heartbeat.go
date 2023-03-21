package main

import (
	"log"
	"time"

	api "github.com/TheCacophonyProject/go-api"
	"github.com/TheCacophonyProject/modemd/connrequester"
	"github.com/TheCacophonyProject/window"
)

const (
	heartBeatDelay = 30 * time.Minute
	interval       = 4 * time.Hour
	attemptDelay   = 5 * time.Second

	connTimeout       = time.Minute * 2
	connRetryInterval = time.Minute * 1
	connMaxRetries    = 3
)

type Heartbeat struct {
	api         *api.CacophonyAPI
	window      *window.Window
	validUntil  time.Time
	end         time.Time
	penultimate bool
	MaxAttempts int
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
	hb := NewHeartbeat(window)
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
	log.Printf("Sending initial heartbeat in %v", initialDelay)
	clock.Sleep(initialDelay)
	for {
		done := hb.updateNextBeat()
		err := sendHeartbeat(hb.validUntil, hb.MaxAttempts)
		if err != nil {
			log.Printf("Error sending heartbeat, skipping this beat %v", err)
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

func NewHeartbeat(window *window.Window) *Heartbeat {
	var nextEnd time.Time
	if !window.NoWindow {
		nextEnd = window.NextEnd()
	}

	h := &Heartbeat{end: nextEnd, window: window, MaxAttempts: 3}
	return h
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

func sendHeartbeat(nextBeat time.Time, attempts int) error {
	cr := connrequester.NewConnectionRequester()
	cr.Start()
	defer cr.Stop()
	if err := cr.WaitUntilUpLoop(connTimeout, connRetryInterval, connMaxRetries); err != nil {
		log.Println("unable to get an internet connection. Not reporting events")
		return err
	}
	var apiClient *api.CacophonyAPI
	var err error
	attempt := 0
	for {
		apiClient, err = api.New()
		if err != nil {
			attempts +=1
			if attempt <= attempts{
				log.Printf("Error connecting to api %v trying again in %v", err, attemptDelay)
				clock.Sleep(attemptDelay)
				continue
			}
			log.Printf("Error connecting to api %v", err)
			return err
		}
		break
	}

	attempt = 0
	for {
		_, err := apiClient.Heartbeat(nextBeat)
		if err == nil {
			log.Printf("Sent heartbeat, valid until %v", nextBeat)
			return nil
		}
		attempt += 1
		if attempt > attempts {
			break
		}
		log.Printf("Error sending heartbeat %v, trying again in %v", err, attemptDelay)
		clock.Sleep(attemptDelay)
	}
	return err
}

func sendFinalHeartBeat(window *window.Window) error {
	log.Printf("Sending final heart beat")
	return sendHeartbeat(window.NextStart().Add(heartBeatDelay*2), 3)
}
