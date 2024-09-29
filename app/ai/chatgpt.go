package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"

	"github.com/go-resty/resty/v2"
)

const (
	apiEndpointChat  = "https://api.openai.com/v1/chat/completions"
	apiEndpointEmbed = "https://api.openai.com/v1/embeddings"
	modelChat        = "gpt-3.5-turbo"
	modelEmbed       = "text-embedding-ada-002"
	modelVision      = "gpt-4o-mini"
)

type Json struct {
	// The JSON object that will be read from the file
	Glossary   string
	Definition string
	Example    string
}

func ReadJsonBytes(jsonBytes []byte) Json {
	var res Json
	err := json.Unmarshal(jsonBytes, &res)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Printf("%+v", res)
	return res
}

func ReadJsonString(json_string string) Json {
	// Function that pases json string
	var res Json
	err := json.Unmarshal([]byte(json_string), &res)
	if err != nil {
		log.Fatal(err)
	}
	return res
}

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

func getEnvAPIKey() string {
	return os.Getenv("API_KEY")
}

type Choice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"` // answer
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

// TODO delete these two functions not used
// Encode image to base64
func encodeImage(image_path string) string {
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
	str := base64.StdEncoding.EncodeToString(buffer)
	return str
}

// Encode image to base64
func encodeImageBytes(picture []byte) string {
	str := base64.StdEncoding.EncodeToString(picture)
	return str
}

func CallVisionApi(question string, image []byte) VisionResponse {
	fmt.Println("Calling vision API")

	client := resty.New()

	// Use a multipart form to send the image and text
	response, err := client.R().
		SetAuthToken(os.Getenv("API_KEY")).
		SetFileReader("file", "image.png", bytes.NewReader(image)). // Pass the image as a file
		SetFormData(map[string]string{
			"model":  modelVision,
			"prompt": question, // Use 'prompt' or 'messages' depending on API
		}).
		Post(apiEndpointChat)

	if err != nil {
		log.Fatalf("Failed to send request: %v\n", err)
	}

	var parsed_response VisionResponse
	err = json.Unmarshal(response.Body(), &parsed_response)
	if err != nil {
		log.Fatalf("Failed to parse response: %v\n", err)
	}

	return parsed_response
}

// TODO not used delete
func StartVision(image []byte) {
	var question string = "Read the text in the image. Extract the word being defined and its definition. If possible also a example of how its used." +
		"Now i want you to respond in the format of json. Key ist the important word and value its definition and example." +
		"I will turn your answer into a json file so please adhere to the format so that i can parse the file easily." +
		"Start with { and end with }. No explaination before or after the json." +
		"And dont format with new line or tabs or spaces. Just the json." +
		` The json will be marshaled into the following go struct:
							type Json struct {
								// The JSON object that will be read from the file
								Glossary   string
								Definition string
								Example    string
							}`
	var response VisionResponse = CallVisionApi(question, image)
	fmt.Printf("%#v\n", response)
}

func ReadPicture(image []byte) Json {
	var question string = "Read the text in the image. Extract the word being defined and its definition. If possible also a example of how its used." +
		"Now i want you to respond in the format of json. Key ist the important word and value its definition and example." +
		"I will turn your answer into a json file so please adhere to the format so that i can parse the file easily." +
		"Start with { and end with }. No explaination before or after the json." +
		"And dont format with new line or tabs or spaces. Just the json." +
		` The json will be marshaled into the following go struct:
							type Json struct {
								// The JSON object that will be read from the file
								Glossary   string
								Definition string
								Example    string
							}`
	var response VisionResponse = CallVisionApi(question, image)
	fmt.Printf("%#v\n", response)
	res := ReadJsonBytes([]byte(response.Choices[0].Message.Content))
	fmt.Printf("%#v\n %v\n %v\n %v\n", res, res.Definition, res.Example, res.Glossary)
	return res
}
