package ai

import (
	"fmt"
	"log"
	"os"
	"smartquiz/app/db"
	"smartquiz/app/types"
	"testing"
)

// Encode image to base64
func fileToBytes(image_path string) []byte {
	file, err := os.Open(image_path)
	if err != nil {
		log.Fatalf("Failed to open file %v\n", err)
	}
	defer file.Close()
	fs, _ := file.Stat()
	size := fs.Size()
	fmt.Println(size)
	buffer := make([]byte, size)
	file.Read(buffer)
	return buffer
}

const ImagePath = `C:\Users\qxz2fxf\Downloads\2024-09-15 21_31_30-glossary _ Übersetzung Englisch-Deutsch.png`

func TestVision(t *testing.T) {
	visionRes := ReadPicture(fileToBytes(ImagePath))
	fmt.Printf("Json succesfully read.\n %v\n%v\n%v", visionRes.Glossary, visionRes.Definition, visionRes.Example)
	germanWord := types.GermanWord{
		DifficultyLevel: "",
		GermanWord:      visionRes.Glossary,
		Definition:      visionRes.Definition,
	}
	err := db.Get().Save(&germanWord).Error
	if err != nil {
		fmt.Println("error when saving glossary to database", err)
		t.Fail()
	}
}
