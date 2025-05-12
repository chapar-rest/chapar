package settings

import (
	"github.com/chapar-rest/chapar/internal/repository"
)

type Controller struct {
	view *View
	repo repository.Repository
}
