package notify

import (
	"sync"
	"time"

	"github.com/mirzakhany/chapar/ui/widgets"
)

var NotificationController = &widgets.Notification{
	Mtx: sync.Mutex{},
}

func Send(text string, duration time.Duration) {
	NotificationController.Mtx.Lock()
	defer NotificationController.Mtx.Unlock()

	NotificationController.EndAt = time.Now().Add(duration)
	NotificationController.Text = text
}
