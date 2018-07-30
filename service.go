package main

import (
	"errors"
	"time"

	"github.com/godbus/dbus"
	"github.com/godbus/dbus/introspect"
)

const (
	dbusName = "org.cacophony.ATtiny"
	dbusPath = "/org/cacophony/ATtiny"
)

type service struct {
	attinyPresent bool
}

func startService(attinyPresent bool) error {
	conn, err := dbus.SystemBus()
	if err != nil {
		return err
	}
	reply, err := conn.RequestName(dbusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		return err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return errors.New("name already taken")
	}

	s := &service{
		attinyPresent: attinyPresent,
	}
	conn.Export(s, dbusPath, dbusName)
	conn.Export(genIntrospectable(s), dbusPath, "org.freedesktop.DBus.Introspectable")
	return nil
}

func genIntrospectable(v interface{}) introspect.Introspectable {
	node := &introspect.Node{
		Interfaces: []introspect.Interface{{
			Name:    dbusName,
			Methods: introspect.Methods(v),
		}},
	}
	return introspect.NewIntrospectable(node)
}

// IsPresent returns whether or not an ATtiny was detected.
func (s service) IsPresent() (bool, *dbus.Error) {
	return s.attinyPresent, nil
}

// StayOnFor will delay turning off the raspberry pi for m minutes.
func (s service) StayOnFor(m int) *dbus.Error {
	err := setStayOnUntil(time.Now().Add(time.Duration(m) * time.Minute))
	if err != nil {
		return &dbus.Error{
			Name: dbusName + ".StayOnForError",
			Body: []interface{}{err.Error()},
		}
	}
	return nil
}
