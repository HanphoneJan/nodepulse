//go:build linux

package main

import (
	"os/exec"

	"github.com/godbus/dbus/v5"
)

func sendNotification(title, body string) error {
	conn, err := dbus.SessionBus()
	if err == nil {
		obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
		call := obj.Call("org.freedesktop.Notifications.Notify", 0,
			"NodePulse", uint32(0), "", title, body,
			[]string{}, map[string]dbus.Variant{"urgency": dbus.MakeVariant(byte(1))}, int32(6000))
		if call.Err == nil {
			return nil
		}
	}
	// Headless sessions or minimal desktops may not expose a notification
	// daemon; notify-send is a best-effort fallback when installed.
	return exec.Command("notify-send", "-a", "NodePulse", "-t", "6000", title, body).Run()
}
