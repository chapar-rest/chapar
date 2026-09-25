package icons

import "github.com/mirzakhany/yoga/render"

func init() {
	for name, label := range badgeLabels {
		render.RegisterIcon(name, badgeSVG(label))
	}
}
