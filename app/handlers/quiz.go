package handlers

import (
	"fmt"
	"smartquiz/app/db"
	"smartquiz/app/views/quiz"

	"smartquiz/app/types"

	"github.com/anthdm/superkit/kit"
)

func GetRandomWord() (types.GermanWord, error) {
	var word types.GermanWord
	err := db.Get().Order("RANDOM()").Limit(1).First(&word).Error
	return word, err
}

func HandleQuizIndex(kit *kit.Kit) error {
	word, err := GetRandomWord()
	if err != nil {
		fmt.Println("unable to get random row")
		return fmt.Errorf("unable to get random row")
	}
	return kit.Render(quiz.Index(word))
}
