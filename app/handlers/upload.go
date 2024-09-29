package handlers

import (
	"fmt"
	"io"
	"net/http"
	"smartquiz/app/ai"
	"smartquiz/app/db"
	"smartquiz/app/types"

	"github.com/anthdm/superkit/kit"
	_ "github.com/mattn/go-sqlite3" // Import SQLite driver
)

func HandleUpload(kit *kit.Kit) error {
	// Parse the multipart form in the request
	fmt.Println("entering handle for upload")
	err := kit.Request.ParseMultipartForm(10 << 20) // limit your max input length!
	if err != nil {
		fmt.Println("error when parsed", err)
		http.Error(kit.Response, err.Error(), http.StatusInternalServerError)
		return err
	}
	fmt.Println("succesfully parsed file")

	// FormFile returns the first file for the given key 'picture'
	file, _, err := kit.Request.FormFile("file")
	if err != nil {
		fmt.Println("error when fromfile", err)
		http.Error(kit.Response, err.Error(), http.StatusInternalServerError)
		return err
	}
	defer file.Close()

	fmt.Println("succesfully fromfile")

	// Read file contents into a byte slice
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		fmt.Println("error when reading file bytes", err)
		http.Error(kit.Response, err.Error(), http.StatusInternalServerError)
		return err
	}

	// Now you have the file as bytes in fileBytes
	fmt.Println("successfully read file into bytes")
	visionRes := ai.ReadPicture(fileBytes)
	germanWord := types.GermanWord{
		DifficultyLevel: "",
		GermanWord:      visionRes.Glossary,
		Definition:      visionRes.Definition,
	}
	err = db.Get().Save(&germanWord).Error
	if err != nil {
		fmt.Println("error when saving glossary to database", err)
		return err
	}
	return kit.Redirect(http.StatusSeeOther, "/track")
}

//	// Create a new file in the current working directory
//	dst, err := os.Create(handler.Filename)
//	if err != nil {
//		fmt.Println("error when creating file", err)
//		http.Error(kit.Response, err.Error(), http.StatusInternalServerError)
//		return err
//	}
//	defer dst.Close()
//	fmt.Println("succesfully creating file")
//
//	// Copy the uploaded file to the destination file
//	if _, err := io.Copy(dst, file); err != nil {
//		fmt.Println("error when saving file", err)
//		http.Error(kit.Response, err.Error(), http.StatusInternalServerError)
//		return err
//	}
//	fmt.Println("succesfully saving file")
//
//	// Optionally, respond back to the client
//	fmt.Fprintf(kit.Response, "File uploaded successfully: %+v", handler.Filename)
//
//	ai.StartVision(handler.Filename)
//	return nil
//}
