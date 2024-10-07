package handlers

import (
	"fmt"
	"net/http"
	"smartquiz/app/db"
	"smartquiz/app/types"
	"smartquiz/app/views/track"

	"github.com/anthdm/superkit/kit"
	_ "github.com/mattn/go-sqlite3" // Import SQLite driver
)

func HandleTrackIndex(kit *kit.Kit) error {
	var germanWords []types.GermanWord
	// var germanWord types.GermanWord
	err := db.Get().Find(&germanWords).Error
	if err != nil {
		fmt.Println("Unable to query glossary from database", err)
		http.Error(kit.Response, "Unable to query glossary from database", http.StatusInternalServerError)
		germanWords = append(germanWords, types.GermanWord{
			Example:    "a",
			GermanWord: "b",
			Definition: "c",
		})
	} else {
		fmt.Println("words: ", germanWords)
		// germanWords = append(germanWords, germanWord)
	}

	// Render the template with the fetched data
	return kit.Render(track.Index(germanWords))
}
