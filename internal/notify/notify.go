package notify

import (
	"time"

	"github.com/mirzakhany/chapar/ui/widgets"
)

var NotificationController = &widgets.Notification{}

func Send(text string, duration time.Duration) {
	NotificationController.EndAt = time.Now().Add(duration)
	NotificationController.Text = text
}
