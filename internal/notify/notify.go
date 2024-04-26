package notify

import (
	"time"

	"github.com/chapar-rest/chapar/ui/widgets"
)

var NotificationController = &widgets.Notification{}

func Send(text string, duration time.Duration) {
	NotificationController.EndAt = time.Now().Add(duration)
	NotificationController.Text = text
}
