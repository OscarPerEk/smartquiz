package handlers

import (
	"smartquiz/app/views/quiz"

	"github.com/anthdm/superkit/kit"
)

func HandleQuizIndex(kit *kit.Kit) error {
	return kit.Render(quiz.Index())
}
