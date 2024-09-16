package ai

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dslipak/pdf"
	"github.com/go-resty/resty/v2"
)

const (
	apiEndpointChat  = "https://api.openai.com/v1/chat/completions"
	apiEndpointEmbed = "https://api.openai.com/v1/embeddings"
	modelChat        = "gpt-3.5-turbo"
	modelEmbed       = "text-embedding-ada-002"
	modelVision      = "gpt-4-vision-preview"
)

// The api key should be saved in a text file in the user's home directory
func getAPIKey() string {
	usr, err := user.Current()
	if err != nil {
		debug.PrintStack()
		log.Fatalf("Error getting user's home directory: %v", err)
	}
	filePath := filepath.Join(usr.HomeDir, ".api_key.txt")
	apiKeyBytes, err := os.ReadFile(filePath)
	if err != nil {
		debug.PrintStack()
		log.Fatalf("Failed to read api key %v\n", err)
	}
	return string(apiKeyBytes)
}

// The embeddings are saved in a json file in the user's home directory
func getEmbeddingsPath() string {
	usr, err := user.Current()
	if err != nil {
		debug.PrintStack()
		log.Fatalf("Error getting user's home directory: %v", err)
	}
	filePath := filepath.Join(usr.HomeDir, "embeddings.json")
	fmt.Println(filePath)
	return filePath
}

// The chatgpt answer is saved in a markdown file in the user's home directory
func getAnswerPath() string {
	usr, err := user.Current()
	if err != nil {
		debug.PrintStack()
		log.Fatalf("Error getting user's home directory: %v", err)
	}
	filePath := filepath.Join(usr.HomeDir, "answer.md")
	return filePath
}

type Config struct {
	Key string `yaml:"openai_api_key"`
}

// Response of OpenAI text completion API
type GptResponse struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created string   `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"` // list of answers/choices
}

type Choice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"` // answer
}

// Response of OpenAI text embedding API
type EmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Model string         `json:"model"`
	Usage map[string]int `json:"usage"`
}

type Embedding struct {
	File     string
	Created  time.Time
	RowStart int
	RowEnd   int
	Vector   []float64
	Content  string
}

type Embeddings struct {
	Created    time.Time
	Embeddings []Embedding
}

// Call embedding from OpenAI API
func CallEmbedding(message string) EmbeddingResponse {
	fmt.Println("Calling Embedding API")
	client := resty.New()
	response, err := client.R().
		SetAuthToken(getAPIKey()).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model":           modelEmbed,
			"input":           message,
			"encoding_format": "float",
		}).
		Post(apiEndpointEmbed)
	if err != nil {
		debug.PrintStack()
		log.Fatalf("Failed to send request %v\n %v\n", err, response)
	}
	var parsedResponse EmbeddingResponse
	json.Unmarshal(response.Body(), &parsedResponse)
	return parsedResponse
}

// Call text completion from OpenAI API
// The system_content is the context of the question
// for example, the content of the file where the question is found
// or instructions on how ChatGPT should answer the question
func CallChatgpt(message string, system_content string) GptResponse {
	fmt.Println("Calling ChatGpt API")
	client := resty.New()
	response, err := client.R().
		SetAuthToken(getAPIKey()).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model": modelChat,
			"messages": []interface{}{
				map[string]interface{}{"role": "system", "content": system_content},
				map[string]interface{}{"role": "user", "content": message},
			},
			"max_tokens": 1000,
		}).
		Post(apiEndpointChat)
	if err != nil {
		debug.PrintStack()
		log.Fatalf("Failed to send request %v\n %v\n", err, response)
	}
	var parsed_response GptResponse
	json.Unmarshal(response.Body(), &parsed_response)
	return parsed_response
}

// Helper function that tells ChatGPT to answer a question based on a context
func CallChatgptWithContext(message string, context string) GptResponse {
	system_content := "Based on the context provided, your job is to first cite the relevant" +
		"answer found in context. Explicitly state in which file the answer is found. Then summarize" +
		"the answer in your own words. Formulate yourself using mark down sytaxt so that your answer can" +
		"be copy pasted to a md file. Your context is:" + context
	return CallChatgpt(message, system_content)
}

// Load embeddings from embeddings.json file in the user's home directory
func LoadEmbeddings() Embeddings {
	jsonFile, _ := os.Open(getEmbeddingsPath())
	defer jsonFile.Close()
	byteContent, _ := io.ReadAll(jsonFile)
	var parsedResponse Embeddings
	json.Unmarshal(byteContent, &parsedResponse)
	return parsedResponse
}

type EmbeddingDistance struct {
	Embedding Embedding
	Distance  float64
}

// L2 norm
func GetVectorDistance(vector1 []float64, vector2 []float64) float64 {
	var distance float64
	for i := 0; i < len(vector1); i++ {
		distance += (vector1[i] - vector2[i]) * (vector1[i] - vector2[i])
	}
	return distance
}

// Sort list of embeddings by distance to the question
func GetEmbeddingDistances(question string, embeddings []Embedding) []EmbeddingDistance {
	var embeddingResponse EmbeddingResponse = CallEmbedding(question)
	var questionEmbedding []float64 = embeddingResponse.Data[0].Embedding
	var distances []EmbeddingDistance
	for _, embedding := range embeddings {
		distances = append(distances, EmbeddingDistance{embedding, GetVectorDistance(embedding.Vector, questionEmbedding)})
	}
	sort.Slice(distances, func(i, j int) bool {
		return distances[i].Distance < distances[j].Distance
	})
	return distances
}

// Turn list of embeddings into a context string
func GetContext(embeddingDistances []EmbeddingDistance, n int) string {
	var context string
	N := min(len(embeddingDistances), n)
	for i := 0; i < N; i++ {
		context += fmt.Sprintf(
			"File: %s\nContent from row %v to row %v:\n%v",
			embeddingDistances[i].Embedding.File,
			embeddingDistances[i].Embedding.RowStart,
			embeddingDistances[i].Embedding.RowEnd,
			embeddingDistances[i].Embedding.Content,
		)
	}
	return context
}

func ReadPdf(path string) string {
	r, err := pdf.Open(path)
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		debug.PrintStack()
		log.Fatalf("Could not extract text from pdf: %v\n", err)
	}
	buf.ReadFrom(b)
	return buf.String()
}

func ReadText(path string) string {
	file, err := os.Open(path)
	if err != nil {
		debug.PrintStack()
		log.Fatalf("Failed to read file %v\n", err)
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var content string
	for scanner.Scan() {
		content += scanner.Text() + "\n"
	}
	return content
}

func ReadFile(path string) string {
	if strings.Contains(path, "https:") {
		// Read file from url
		client := resty.New()
		response, err := client.R().Get(path)
		if err != nil {
			debug.PrintStack()
			log.Fatalf("Failed to send request %v\n %v\n", err, response)
		}
		return response.String()
	}
	split := strings.Split(path, ".")
	if len(split) > 1 {
		ftype := split[len(split)-1]
		if ftype == "pdf" {
			content := ReadPdf(path)
			return content
		}
	}
	return ReadText(path)
}

// Convert a file to embeddings by reading the file to string,
// then splitting the text into chunks of 200 lines
// and then calling the OpenAI text embedding API
func ConvertFileToEmbeddings(path string) ([]Embedding, error) {
	content := ReadFile(path)
	if strings.Contains(content, "#protected") {
		return nil, fmt.Errorf("file is protected")
	}
	var embeddings []Embedding
	var rowStart int
	var rowEnd int
	var lines []string = strings.Split(content, "\n")
	for i := 0; i < len(lines); i += 200 {
		if i-50 >= 0 {
			rowStart = i - 50
		} else {
			rowStart = i
		}
		if i+250 <= len(lines) {
			rowEnd = i + 250
		} else {
			rowEnd = len(lines)
		}
		contentPart := strings.Join(lines[rowStart:rowEnd], "\n")
		var embeddingResponse EmbeddingResponse = CallEmbedding(contentPart)
		if len(embeddingResponse.Data) > 0 {
			embeddings = append(
				embeddings,
				Embedding{
					File:     path,
					Created:  time.Now(),
					RowStart: rowStart,
					RowEnd:   rowEnd,
					Vector:   embeddingResponse.Data[0].Embedding,
					Content:  contentPart,
				},
			)
		}
	}
	return embeddings, nil
}

// Save embeddings to embeddings.json file in the user's home directory
func SaveEmbeddings(embeddings []Embedding) {
	var embeddingsToSave Embeddings = Embeddings{
		Created:    time.Now(),
		Embeddings: embeddings,
	}
	embeddingsJson, err := json.Marshal(embeddingsToSave)
	if err != nil {
		debug.PrintStack()
		log.Fatalf("Failed to marshal embeddings to json %v\n", err)
	}
	err = os.WriteFile(getEmbeddingsPath(), embeddingsJson, 0644)
	if err != nil {
		debug.PrintStack()
		log.Fatalf("Failed to write embeddings to file %v\n", err)
	}
}

func WriteAnswerToFile(response GptResponse, embedding Embedding) {
	// Write answer to file
	answer := "# Answer from " + modelChat + "\n\n" + response.Choices[0].Message.Content +
		"\n\n# Matched Context: \nFile: " + embedding.File +
		"\nRow Start: " + fmt.Sprintf("%v", embedding.RowStart) +
		"\nRow End: " + fmt.Sprintf("%v", embedding.RowEnd) +
		"\n\n" + embedding.Content
	file, err := os.Create(getAnswerPath())
	if err != nil {
		log.Fatalf("Failed to create file %v\n", err)
	}
	defer file.Close()
	_, err = file.WriteString(answer)
	if err != nil {
		log.Fatalf("Failed to write to file %v\n", err)
	}
}

// Helper function that tries to convert file to embeddings
// and sends the embeddings to the embeddings channel.
// It prints a success or fail message to the console.
func EmbedFile(path string, embeddingsChannel chan []Embedding) {
	fmt.Println("\nFound file: ", path)
	fmt.Println("\nCreating new embedding: ", path)
	newEmbeddings, err := ConvertFileToEmbeddings(path)
	if err != nil {
		fmt.Printf("\nFailed to create embedding: %v\n: %v\n", path, err)
	} else {
		fmt.Println("\nSuccessfully created embedding")
		embeddingsChannel <- newEmbeddings
	}
}

// Checks if path is a website, file or folder. Then embedds its content.
// If folder, we recursively embed all files in the folder.
// Each file is embedded in a separate go routine.
func EmbedAnything(path string, wg *sync.WaitGroup, embeddingsChannel chan []Embedding) {
	// website
	if strings.Contains(path, "https:") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			EmbedFile(path, embeddingsChannel)
		}()
		return
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Println("Error:", err)
	} else if fileInfo.Mode().IsRegular() {
		// file
		wg.Add(1)
		go func() {
			defer wg.Done()
			EmbedFile(path, embeddingsChannel)
		}()
	} else if fileInfo.Mode().IsDir() {
		// folder
		fmt.Println("Found folder")
		fmt.Println(path, "is a directory.")

		folder, err := os.Open(path)
		if err != nil {
			fmt.Printf("Error loading folder: %v\n%v\n", path, err)
		}
		fileInfos, err := folder.Readdir(-1)
		folder.Close()
		if err != nil {
			fmt.Printf("Error reading folder: %v\n%v\n", path, err)
		}
		for _, fileInfo := range fileInfos {
			p := filepath.Join(path, fileInfo.Name())
			EmbedAnything(p, wg, embeddingsChannel)
		}
	}
}

// Starting point for creating embeddings from a path.
// The embeddings are saved to the user's home directory.
func StartEmbedding(path string) {
	var wg sync.WaitGroup
	var embeddingsChannel chan []Embedding = make(chan []Embedding)
	EmbedAnything(
		path,
		&wg,
		embeddingsChannel,
	)
	var newEmbeddings []Embedding
	go func() {
		for r := range embeddingsChannel {
			newEmbeddings = append(newEmbeddings, r...)
		}
	}()
	wg.Wait()
	close(embeddingsChannel)
	// Get previous embeddings
	fmt.Println("\nLoading previous embeddings: ", getEmbeddingsPath())
	embeddings := LoadEmbeddings().Embeddings
	embeddings = append(embeddings, newEmbeddings...)
	fmt.Println("\nSaving embedddings")
	SaveEmbeddings(embeddings)
}

// Starting point for asking ChatGPT a question based
// on the best matching context from embeddings
// saved in the user's home directory.
func StartChat(question string) {
	var embeddings []Embedding = LoadEmbeddings().Embeddings
	var embeddingDistances []EmbeddingDistance = GetEmbeddingDistances(question, embeddings)
	var context string = GetContext(embeddingDistances, 2)
	var response GptResponse = CallChatgptWithContext(question, context)
	fmt.Printf("Answer from %v \n\n%v", modelChat, response.Choices[0].Message.Content)
	WriteAnswerToFile(response, embeddingDistances[0].Embedding)
}

// Save the API key to a text file in the user's home directory
func WriteAPIKey(apiKey string) {
	usr, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting user's home directory: %v", err)
	}
	filePath := filepath.Join(usr.HomeDir, ".api_key.txt")
	err = os.WriteFile(filePath, []byte(apiKey), 0644)
	if err != nil {
		log.Fatalf("Failed to write api key %v\n", err)
	}
}

// Response of OpenAI vision API
type Usage struct {
	PromtTokens      int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
type VisionResponse struct {
	Id      string   `json:"id"`
	Object  string   `json:"object"`
	Created string   `json:"created"`
	Model   string   `json:"model"`
	Usage   Usage    `json:"usage"`
	Choices []Choice `json:"choices"`
}

// Encode image to base64
func encodeImage(image_path string) string {
	file, err := os.Open(image_path)
	if err != nil {
		log.Fatalf("Failed to open file %v\n", err)
	}
	defer file.Close()
	fs, _ := file.Stat()
	size := fs.Size()
	buffer := make([]byte, size)
	file.Read(buffer)
	str := base64.StdEncoding.EncodeToString(buffer)
	return str
}

// Ask a question about an image by calling the vision API
func CallVisionApi(question string, image_path string) VisionResponse {
	fmt.Println("Calling vision API")
	client := resty.New()
	response, err := client.R().
		SetAuthToken(getAPIKey()).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"model": modelVision,
			"messages": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]interface{}{
						{"type": "text", "text": question},
						{"type": "image_url", "image_url": map[string]string{
							// "url": "data:image/jpeg;base64," + encodeImage(image_path),
							"url": "data:image/png;base64," + encodeImage(image_path),
						}},
					},
				},
			},
			"max_tokens": 1000,
		}).
		Post(apiEndpointChat)
	if err != nil {
		log.Fatalf("Failed to send request %v\n %v\n", err, response)
	}
	var parsed_response VisionResponse
	json.Unmarshal(response.Body(), &parsed_response)
	return parsed_response
}

func StartVision(image_path string) {
	var question string = "Read the text in the image. Extract the word being defined and its definition. If possible also a example of how its used."
	var response VisionResponse = CallVisionApi(question, image_path)
	fmt.Printf("%#v\n", response)
}

// Parse user input and either:
// 1. Add an api key to the system with flag --key
// 2. Embedd a file or folder with flag --embed
// 3. Extract text from picture with flag --vision
// 4. Ask ChatGPT a question
func main() {
	var embedPath string
	var visionPath string
	var apiKey string
	flag.StringVar(&embedPath, "embed", "", "Embedd a file or folder")
	flag.StringVar(&visionPath, "vision", "", "Extract text from picture")
	flag.StringVar(&apiKey, "key", "", "Add an api key to the system")
	flag.Parse()
	args := flag.Args()
	if embedPath != "" {
		StartEmbedding(embedPath)
	} else if visionPath != "" {
		StartVision(visionPath)
	} else if apiKey != "" {
		WriteAPIKey(apiKey)
	} else {
		StartChat(args[0])
	}
}
