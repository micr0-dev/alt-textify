package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type AltTextResponse struct {
	AltTexts []string `json:"alt_texts"`
}

func runOllamaCommand(imagePath string, model string) (string, error) {
	cmd := exec.Command("ollama", "run", model, fmt.Sprintf("You are an assistant for the visually impaired. Answer concisely for someone who is visually impaired. Write an alt-text for this image. Your response should be one or two sentences. Just state what you see descriptively. %s", imagePath))

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func parseOutput(output string) string {
	patterns := []string{
		`"(.*?)"`,
		`Added image '.*?'\n (.*?)$`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(output); len(matches) > 1 {
			return matches[1]
		}
	}

	return strings.TrimSpace(output)
}

func altTextHandler(w http.ResponseWriter, r *http.Request) {
	imagePath := r.URL.Query().Get("image_path")
	count := 3
	model := r.URL.Query().Get("model")

	if cnt := r.URL.Query().Get("count"); cnt != "" {
		fmt.Sscanf(cnt, "%d", &count)
	}

	if model == "" {
		model = "llava"
	}

	if imagePath == "" {
		http.Error(w, "image_path is required", http.StatusBadRequest)
		return
	}

	var altTexts []string

	for i := 0; i < count; i++ {
		output, err := runOllamaCommand(imagePath, model)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error executing command: %v", err), http.StatusInternalServerError)
			return
		}

		altText := parseOutput(output)
		if altText != "" {
			altTexts = append(altTexts, altText)
		}
	}

	response := AltTextResponse{AltTexts: altTexts}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	count := flag.Int("count", 3, "Number of alt texts to generate. Default is 3.")
	help := flag.Bool("help", false, "Show usage information.")
	server := flag.Bool("server", false, "Run as a web server.")
	port := flag.String("port", "8080", "Port to run the server on. Default is 8080.")
	model := flag.String("model", "llava", "Model to use for generating alt text. Default is llava.")

	flag.Usage = func() {
		fmt.Println("Usage:")
		fmt.Printf("  %s [options] <image_path>\n", os.Args[0])
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nPositional Arguments:")
		fmt.Println("  image_path   Path to the image file.")
	}

	flag.Parse()

	if *help || (*server == false && flag.NArg() == 0) {
		flag.Usage()
		if flag.NArg() == 0 && *server == false {
			fmt.Println("\nError: Image path is required.")
		}
		return
	}

	if *server {
		if _, err := fmt.Sscanf(*port, "%d", new(int)); err != nil {
			fmt.Println("Invalid port number.")
			return
		}

		http.HandleFunc("/generate-alt-text", altTextHandler)
		fmt.Println("Running server on :", *port)
		if err := http.ListenAndServe(":"+*port, nil); err != nil {
			fmt.Println("Error starting server:", err)
		}
	} else {
		imagePath := flag.Arg(0)
		var altTexts []string

		for i := 0; i < *count; i++ {
			output, err := runOllamaCommand(imagePath, *model)
			if err != nil {
				fmt.Println("Error executing command:", err)
				return
			}

			altText := parseOutput(output)
			if altText != "" {
				altTexts = append(altTexts, altText)
			}
		}

		if len(altTexts) > 0 {
			fmt.Println("Generated Alt Texts:")
			for i, altText := range altTexts {
				fmt.Printf("%d. %s\n", i+1, altText)
			}
		} else {
			fmt.Println("No alt texts generated.")
		}
	}
}
