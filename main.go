package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	gowiki "github.com/trietmn/go-wiki"
	"github.com/w0ikid/wiki2docx/internal/docx"
	"github.com/w0ikid/wiki2docx/internal/wiki"
)

func main() {
	inputFile := flag.String("input", "", "Path to a .txt file with article titles (one per line)")
	randomN := flag.Int("random", 1, "Number of random articles to fetch (used when -input is not set)")
	outDir := flag.String("out", "./output", "Directory to save DOCX files")
	workers := flag.Int("workers", 5, "Number of concurrent workers")
	lang := flag.String("lang", "en", "Wikipedia language prefix (e.g. en, ru, de)")
	flag.Parse()

	gowiki.SetLanguage(*lang)
	// go-wiki defaults to http:// which gets redirected → force https
	gowiki.SetURL("https://%v.wikipedia.org/w/api.php")
	gowiki.SetUserAgent("wiki2docx/1.0 (github.com/w0ikid/wiki2docx)")

	// --- Collect titles ---
	titles, err := collectTitles(*inputFile, *randomN)
	if err != nil {
		log.Fatalf("Failed to collect titles: %v", err)
	}
	if len(titles) == 0 {
		log.Fatal("No article titles found. Use -input or -random flags.")
	}

	fmt.Printf("Processing %d article(s) with %d worker(s)...\n", len(titles), *workers)

	// --- Worker pool ---
	titlesCh := make(chan string, len(titles))
	for _, t := range titles {
		titlesCh <- t
	}
	close(titlesCh)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed []string

	for i := 0; i < *workers; i++ {
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
