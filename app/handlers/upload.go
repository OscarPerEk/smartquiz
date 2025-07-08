package handlers

import (
	"fmt"
	"io"
	"net/http"
	"smartquiz/app/ai"
	"smartquiz/app/db"
	"smartquiz/app/types"
	"sync"

	"github.com/anthdm/superkit/kit"
	_ "github.com/mattn/go-sqlite3" // Import SQLite driver
	"mime/multipart"
)

// HandleUpload processes multiple uploaded files in parallel.
// For each file, it reads the content, sends it to an AI service (OpenAI API call).
// All successfully processed results are then saved to the database in a single batch operation
// using a GORM transaction to ensure atomicity.
// This version emphasizes clear, basic Go concurrency principles with an added batch save.
func HandleUpload(kit *kit.Kit) error {
	fmt.Println("Starting the file upload handling process.")

	// Step 1: Parse the incoming HTTP request to get the uploaded files.
	// We limit the total size of the multipart form to 10MB (10 << 20 bytes).
	err := kit.Request.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("Error parsing multipart form from request:", err)
		http.Error(kit.Response, "Failed to parse form: "+err.Error(), http.StatusInternalServerError)
		return err
	}
	fmt.Println("Successfully parsed the multipart form data.")

	// Step 2: Extract the files from the "files" field in the form.
	files := kit.Request.MultipartForm.File["files"]
	if len(files) == 0 {
		fmt.Println("No files were found in the 'files' field of the form.")
		return fmt.Errorf("no files provided in the form")
	}

	// --- Concurrency Management Setup ---

	// WaitGroup: This is used to make sure the main function waits until ALL
	// the background tasks (goroutines) for processing files are finished.
	var wg sync.WaitGroup

	// firstProcessingError: This variable will store the very first error encountered
	// by any of our parallel goroutines during file reading or AI processing.
	var firstProcessingError error

	// once: This ensures that `firstProcessingError` is set only once,
	// even if multiple goroutines encounter errors simultaneously.
	var once sync.Once

	// resultsChan: A channel to collect `types.GermanWord` results from each
	// successfully processed file. It's buffered with the number of files
	// to avoid blocking goroutines sending results.
	resultsChan := make(chan types.GermanWord, len(files)*100)

	// --- Parallel File Processing Loop ---

	// Loop through each uploaded file to start a new, independent background task (goroutine).
	for _, fileHeader := range files {
		wg.Add(1) // Increment the WaitGroup counter for each new goroutine.

		// Launch a new goroutine for concurrent file processing.
		// Pass `fileHeader` as an argument (`fh`) to create a unique copy for each goroutine,
		// preventing race conditions with loop variables.
		go func(fh *multipart.FileHeader) {
			defer wg.Done() // Decrement the WaitGroup counter when this goroutine finishes.

			fmt.Printf("Processing started for file: %s\n", fh.Filename)

			// Helper function: Records the first encountered processing error.
			recordProcessingError := func(err error) {
				once.Do(func() {
					firstProcessingError = err // Capture the very first error.
				})
			}

			// --- Task 1: Open the uploaded file ---
			file, err := fh.Open()
			if err != nil {
				recordProcessingError(fmt.Errorf("failed to open file %s: %w", fh.Filename, err))
				fmt.Printf("Error: Could not open file %s: %v\n", fh.Filename, err)
				return // Stop processing this specific file.
			}
			defer file.Close() // Ensure the file is closed when this goroutine finishes.

			// --- Task 2: Read the file's content into a byte slice ---
			fileBytes, err := io.ReadAll(file)
			if err != nil {
				recordProcessingError(fmt.Errorf("failed to read bytes from file %s: %w", fh.Filename, err))
				fmt.Printf("Error: Could not read bytes from file %s: %v\n", fh.Filename, err)
				return
			}
			fmt.Printf("Successfully read bytes from file %s.\n", fh.Filename)

			// --- Task 3: Call the AI service (e.g., OpenAI API) ---
			// This is the part that benefits most from parallelization.
			visionRes := ai.ReadPicture(fileBytes)
			fmt.Printf("AI successfully processed file %s.\n", fh.Filename)

			// --- Task 4: Map the AI response to our data structure ---
			for _, word := range visionRes {
				germanWord := types.GermanWord{
					Example:     word.Example,
					GermanWord:  word.Glossary,
					Definition:  word.Definition,
					Translation: word.Translations,
				}

				// Send the successfully processed word to the results channel.
				resultsChan <- germanWord
				fmt.Printf("Result for %s sent to collection channel.\n", fh.Filename)
			}
		}(fileHeader) // Pass the `fileHeader` for the current iteration to the goroutine.
	}

	// --- Final Synchronization and Result Collection ---

	// This blocks the main function until all started goroutines have completed.
	wg.Wait()
	// Close the results channel after all goroutines are done sending.
	// This signals to the `for range` loop below that no more values will be sent.
	close(resultsChan)

	// Collect all successful results from the channel into a slice.
	var wordsToSave []types.GermanWord
	for word := range resultsChan {
		wordsToSave = append(wordsToSave, word)
	}
	fmt.Printf("Collected %d words for batch saving.\n", len(wordsToSave))

	// --- Error Check for Parallel Processing Phase ---

	// If any error occurred during the parallel file reading or AI processing, return it.
	if firstProcessingError != nil {
		fmt.Printf("Overall file processing failed due to an error: %v\n", firstProcessingError)
		http.Error(kit.Response, "Error during file processing: "+firstProcessingError.Error(), http.StatusInternalServerError)
		return firstProcessingError
	}

	// --- Batch Database Save with Transaction ---

	// Only attempt to save if there are words to save.
	if len(wordsToSave) > 0 {
		fmt.Println("Attempting to save all collected words to the database in a batch using a transaction.")

		// Get the GORM DB instance.
		dbInstance := db.Get()
		if dbInstance == nil {
			err := fmt.Errorf("database instance is nil")
			fmt.Println("Error:", err)
			http.Error(kit.Response, err.Error(), http.StatusInternalServerError)
			return err
		}

		// Start a database transaction.
		tx := dbInstance.Begin()
		if tx.Error != nil {
			fmt.Printf("Error starting database transaction: %v\n", tx.Error)
			http.Error(kit.Response, "Error starting database transaction: "+tx.Error.Error(), http.StatusInternalServerError)
			return tx.Error
		}

		// Iterate through each word and save it within the transaction.
		for _, word := range wordsToSave {
			if saveErr := tx.Save(&word).Error; saveErr != nil {
				// If any save fails, roll back the entire transaction.
				tx.Rollback()
				fmt.Printf("Error saving word '%s' in transaction: %v. Rolling back transaction.\n", word.GermanWord, saveErr)
				http.Error(kit.Response, "Error saving words to database: "+saveErr.Error(), http.StatusInternalServerError)
				return saveErr
			}
		}

		// If all saves within the loop succeed, commit the transaction.
		if commitErr := tx.Commit().Error; commitErr != nil {
			fmt.Printf("Error committing database transaction: %v\n", commitErr)
			http.Error(kit.Response, "Error committing database transaction: "+commitErr.Error(), http.StatusInternalServerError)
			return commitErr
		}

		fmt.Println("Successfully saved all words to the database within a transaction.")
	} else {
		fmt.Println("No words were successfully processed to save to the database.")
	}

	fmt.Println("All file processing and database saving operations completed.")
	// If everything succeeded, redirect the user to the "/track" page.
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
