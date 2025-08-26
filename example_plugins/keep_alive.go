package plugins

import (
	"gt"
	"time"
)

const maxKeepAlive = 5

const PluginName = "keep_alive"

var (
	keepAliveCount = 0
	lastKeepalive  time.Time
)

func init() {
	gt.RegisterChatHandler(watchChat)
	lastKeepalive = time.Now()
}

func watchChat(msg string) {
	if msg == "Please do something to avoid being disconnected." {
		if time.Since(lastKeepalive) < time.Minute*2 {
			//Way too soon
			return
		}
		if time.Since(lastKeepalive) > time.Minute*30 {
			//Its been long enough.. User is not AFK reset the count
			keepAliveCount = 0
		}
		if keepAliveCount > maxKeepAlive {
			//Don't prevent disconnect forever
			return
		}
		gt.EnqueueCommand("/money")
		lastKeepalive = time.Now()
		keepAliveCount++
	}
}
