//go:build linux

package main

import "os"

// tuneWebKit works around slow scrolling with WebKitGTK's DMA-BUF renderer on
// some Wayland/Intel setups. Users can opt back in with
// NODEPULSE_WEBKIT_DMABUF=1.
func tuneWebKit() {
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return
	}
	if os.Getenv("NODEPULSE_WEBKIT_DMABUF") == "1" {
		return
	}
	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
}
