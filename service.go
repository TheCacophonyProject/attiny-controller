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

type service struct{}

// StartService starts attiny controller dbus service
func StartService() error {
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

	s := &service{}
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

// StayOnFor will delay turning off the raspberry pi for m minutes.
func (s service) StayOnFor(m int) *dbus.Error {
	err := SetStayOnUntil(time.Now().Add(time.Duration(m) * time.Minute))
	if err != nil {
		return &dbus.Error{
			Name: dbusName + ".StayOnForError",
			Body: []interface{}{err.Error()},
		}
	}
	return nil
}
