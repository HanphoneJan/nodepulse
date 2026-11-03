//go:build windows

package main

import toast "git.sr.ht/~jackmordaunt/go-toast/v2"

func sendNotification(title, body string) error {
	n := toast.Notification{
		AppID: "NodePulse",
		Title: title,
		Body:  body,
	}
	return n.Push()
}
