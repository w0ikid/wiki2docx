package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/w0ikid/wiki2docx/internal/docx"
	"github.com/w0ikid/wiki2docx/internal/wiki"
)

func main() {
	var (
		inputFile = flag.String("input", "", "Path to a .txt file with article titles (one per line)")
		randomN   = flag.Int("random", 1, "Number of random articles to fetch (used when -input is not set)")
		lang      = flag.String("lang", "en", "Wikipedia language prefix (e.g. en, ru, de)")
		nWorkers  = flag.Int("workers", 5, "Number of concurrent workers")
		outDir    = flag.String("out", "./output", "Directory to save DOCX files")

		// Aliases
		workerAlias = flag.Int("worker", 0, "Alias for -workers")
		wAlias      = flag.Int("w", 0, "Short alias for -workers")
		outputAlias = flag.String("output", "", "Alias for -out")
		oAlias      = flag.String("o", "", "Short alias for -out")
		rateLimit   = flag.Int("rate", 10, "Global rate limit in requests per second (0 = no limit)")
	)
	flag.Parse()

	// Merge aliases
	if *workerAlias != 0 {
		*nWorkers = *workerAlias
	}
	if *wAlias != 0 {
		*nWorkers = *wAlias
	}
	if *outputAlias != "" {
		*outDir = *outputAlias
	}
	if *oAlias != "" {
		*outDir = *oAlias
	}

	// Increase the number of idle connections per host to allow true parallelism.
	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport.MaxIdleConnsPerHost = *nWorkers + 2
	}

	// Apply settings to our custom wiki package.
	wiki.SetLanguage(*lang)
	wiki.SetRateLimit(*rateLimit)

	// Set a reasonable timeout for HTTP requests to prevent hangs.
	http.DefaultClient.Timeout = 30 * time.Second

	// --- Collect titles ---
	fmt.Printf("Collecting article titles (random: %d, lang: %s)...\n", *randomN, *lang)
	titles, err := collectTitles(*inputFile, *randomN)
	if err != nil {
		log.Fatalf("Failed to collect titles: %v", err)
	}
	if len(titles) == 0 {
		log.Fatal("No article titles found. Use -input or -random flags.")
	}

	fmt.Printf("Processing %d article(s) with %d worker(s)...\n", len(titles), *nWorkers)

	// --- Worker pool ---
	titlesCh := make(chan string, len(titles))
	for _, t := range titles {
		titlesCh <- t
	}
	close(titlesCh)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed []string

	for i := 0; i < *nWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for title := range titlesCh {
				if err := processArticle(title, *outDir); err != nil {
					mu.Lock()
					failed = append(failed, fmt.Sprintf("%s: %v", title, err))
					mu.Unlock()
					fmt.Printf("  [FAIL] %s: %v\n", title, err)
				} else {
					fmt.Printf("  [OK]   %s\n", title)
				}
			}
		}()
	}

	wg.Wait()

	fmt.Printf("\nDone. %d succeeded, %d failed.\n", len(titles)-len(failed), len(failed))
	for _, f := range failed {
		fmt.Println("  ERROR:", f)
	}
}

// collectTitles returns a list of article titles either from a file or from Wikipedia's random endpoint.
func collectTitles(inputFile string, randomN int) ([]string, error) {
	if inputFile != "" {
		return readTitlesFromFile(inputFile)
	}
	return wiki.GetRandomTitles(randomN)
}

// readTitlesFromFile reads non-empty, non-comment lines from a text file.
func readTitlesFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var titles []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		titles = append(titles, line)
	}
	return titles, scanner.Err()
}

// processArticle fetches a Wikipedia article and writes it to a DOCX file.
func processArticle(title, outDir string) error {
	article, err := wiki.FetchArticle(title)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	if err := docx.Build(article.Title, article.Content, outDir); err != nil {
		return fmt.Errorf("build docx: %w", err)
	}
	return nil
}
