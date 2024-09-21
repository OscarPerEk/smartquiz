package handlers

import (
	"smartquiz/app/views/upload"

	"github.com/anthdm/superkit/kit"
)

func HandleUploadIndex(kit *kit.Kit) error {
	return kit.Render(upload.Index())
}
