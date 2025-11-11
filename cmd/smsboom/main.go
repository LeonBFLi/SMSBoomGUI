package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type requestDefinition struct {
	Name    string            `json:"name"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type requestFile struct {
	Requests []requestDefinition `json:"requests"`
}

type task struct {
	definition requestDefinition
	requestIdx int
	iteration  int
}

func main() {
	phone := flag.String("p", "", "Phone number to target")
	apiPath := flag.String("a", "", "Path to the API definition file")
	workers := flag.Int("c", 4, "Number of concurrent workers")
	iterations := flag.Int("n", 1, "How many times to execute the full API list")
	delay := flag.Duration("delay", 0, "Delay between individual requests per worker")
	timeout := flag.Duration("timeout", 10*time.Second, "HTTP client timeout")
	dryRun := flag.Bool("dry-run", false, "Print the prepared requests without executing them")
	verbose := flag.Bool("v", false, "Print detailed progress information")
	placeholder := flag.String("placeholder", "{{phone}}", "Placeholder token in the API file that should be replaced with the phone number")

	flag.Parse()

	if strings.TrimSpace(*phone) == "" {
		exitWithError(errors.New("missing required -p phone number"))
	}

	if strings.TrimSpace(*apiPath) == "" {
		exitWithError(errors.New("missing required -a API file path"))
	}

	if *workers < 1 {
		exitWithError(fmt.Errorf("invalid worker count %d", *workers))
	}

	if *iterations < 1 {
		exitWithError(fmt.Errorf("invalid iteration count %d", *iterations))
	}

	definitions, err := loadDefinitions(*apiPath)
	if err != nil {
		exitWithError(err)
	}

	if len(definitions) == 0 {
		exitWithError(errors.New("no API requests defined in the provided file"))
	}

	if *dryRun {
		printDryRun(definitions, *phone, *placeholder)
		return
	}

	client := &http.Client{Timeout: *timeout}

	tasks := make(chan task)
	var wg sync.WaitGroup
	var successCount int64
	var failureCount int64

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for t := range tasks {
				err := executeRequest(client, t, *phone, *placeholder)
				if err != nil {
					atomic.AddInt64(&failureCount, 1)
					fmt.Fprintf(os.Stderr, "[worker %d] iteration %d request %d failed: %v\n", workerID, t.iteration, t.requestIdx, err)
				} else {
					atomic.AddInt64(&successCount, 1)
					if *verbose {
						fmt.Printf("[worker %d] iteration %d request %d succeeded\n", workerID, t.iteration, t.requestIdx)
					}
				}

				if *delay > 0 {
					time.Sleep(*delay)
				}
			}
		}(i + 1)
	}

	start := time.Now()
	for iteration := 1; iteration <= *iterations; iteration++ {
		for idx, def := range definitions {
			tasks <- task{definition: def, requestIdx: idx + 1, iteration: iteration}
		}
	}
	close(tasks)
	wg.Wait()

	duration := time.Since(start)
	totalRequests := len(definitions) * (*iterations)
	fmt.Printf("Completed %d iterations (%d total requests) in %s\n", *iterations, totalRequests, duration.Round(time.Millisecond))
	fmt.Printf("Success: %d\n", successCount)
	fmt.Printf("Failed: %d\n", failureCount)

	if failureCount > 0 {
		os.Exit(1)
	}
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

func loadDefinitions(path string) ([]requestDefinition, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open API file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read API file: %w", err)
	}

	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})

	// try decoding as array first
	var arrayDefs []requestDefinition
	if err := json.Unmarshal(data, &arrayDefs); err == nil && len(arrayDefs) > 0 {
		return arrayDefs, nil
	}

	var fileDefs requestFile
	if err := json.Unmarshal(data, &fileDefs); err != nil {
		return nil, fmt.Errorf("invalid API file format: %w", err)
	}

	return fileDefs.Requests, nil
}

func executeRequest(client *http.Client, t task, phone, placeholder string) error {
	method := strings.ToUpper(strings.TrimSpace(t.definition.Method))
	if method == "" {
		method = http.MethodGet
	}

	url := replacePhonePlaceholder(t.definition.URL, phone, placeholder)
	if strings.TrimSpace(url) == "" {
		return errors.New("request is missing a URL")
	}

	var body io.Reader
	if strings.TrimSpace(t.definition.Body) != "" {
		bodyStr := replacePhonePlaceholder(t.definition.Body, phone, placeholder)
		body = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	for key, value := range t.definition.Headers {
		req.Header.Set(key, replacePhonePlaceholder(value, phone, placeholder))
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "smsboom-linux/1.0")
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("execute HTTP request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP status %s", resp.Status)
	}

	return nil
}

func replacePhonePlaceholder(value, phone, placeholder string) string {
	phone = strings.TrimSpace(phone)
	if value == "" {
		return value
	}

	replacements := []string{placeholder, strings.ToUpper(placeholder), strings.ToLower(placeholder), "{phone}", "{PHONE}", "%phone%", "%PHONE%", "{{mobile}}", "{{MOBILE}}"}
	result := value
	for _, token := range replacements {
		result = strings.ReplaceAll(result, token, phone)
	}
	return result
}

func printDryRun(defs []requestDefinition, phone, placeholder string) {
	fmt.Printf("Loaded %d API request(s). Dry-run output:\n", len(defs))
	for idx, def := range defs {
		method := def.Method
		if method == "" {
			method = http.MethodGet
		}
		url := replacePhonePlaceholder(def.URL, phone, placeholder)
		body := replacePhonePlaceholder(def.Body, phone, placeholder)
		fmt.Printf("%d. %s %s\n", idx+1, strings.ToUpper(method), url)
		if len(def.Headers) > 0 {
			fmt.Println("   Headers:")
			for key, value := range def.Headers {
				fmt.Printf("     %s: %s\n", key, replacePhonePlaceholder(value, phone, placeholder))
			}
		}
		if strings.TrimSpace(body) != "" {
			fmt.Printf("   Body: %s\n", body)
		}
	}
}
