package wiki

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Article holds the fetched data for a Wikipedia article.
type Article struct {
	Title   string
	Content string
}

// langPrefix is a package-level variable to store the Wikipedia language prefix.
var langPrefix = "en"

// rateLimiter is a channel used for global rate limiting across the package.
var rateLimiter <-chan time.Time

// SetLanguage sets the Wikipedia language prefix (e.g., "en", "ru").
func SetLanguage(l string) {
	langPrefix = l
}

// SetRateLimit initializes the global rate limiter with the given requests per second.
// If rps is 0 or less, no rate limiting is applied.
func SetRateLimit(rps int) {
	if rps <= 0 {
		rateLimiter = nil
		return
	}
	rateLimiter = time.Tick(time.Second / time.Duration(rps))
}

func wait() {
	if rateLimiter != nil {
		<-rateLimiter
	}
}

// wikiClient is a shared http client for all wiki package requests.
var wikiClient = &http.Client{}

// FetchArticle retrieves the full text of a Wikipedia article by title.
func FetchArticle(title string) (Article, error) {
	apiURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php", langPrefix)
	params := url.Values{}
	params.Set("action", "query")
	params.Set("prop", "extracts")
	params.Set("explaintext", "1")
	params.Set("titles", title)
	params.Set("format", "json")
	params.Set("redirects", "1")

	wait()
	req, err := http.NewRequest("GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return Article{}, err
	}
	req.Header.Set("User-Agent", "wiki2docx/1.0 (github.com/w0ikid/wiki2docx)")

	resp, err := wikiClient.Do(req)
	if err != nil {
		return Article{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Article{}, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var res struct {
		Query struct {
			Pages map[string]struct {
				Title   string `json:"title"`
				Extract string `json:"extract"`
			} `json:"pages"`
		} `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return Article{}, err
	}

	for _, page := range res.Query.Pages {
		return Article{
			Title:   page.Title,
			Content: page.Extract,
		}, nil
	}

	return Article{}, fmt.Errorf("article not found: %s", title)
}

// GetRandomTitles returns up to limit unique random Wikipedia article titles.
func GetRandomTitles(limit int) ([]string, error) {
	apiURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php", langPrefix)
	uniqueTitles := make(map[string]struct{})
	var result []string

	maxAttempts := limit * 3
	attempts := 0

	for len(uniqueTitles) < limit && attempts < maxAttempts {
		attempts++
		batchSize := limit - len(uniqueTitles)
		if batchSize > 500 {
			batchSize = 500
		}

		params := url.Values{}
		params.Set("action", "query")
		params.Set("list", "random")
		params.Set("rnnamespace", "0")
		params.Set("rnlimit", fmt.Sprintf("%d", batchSize))
		params.Set("format", "json")

		wait()
		req, err := http.NewRequest("GET", apiURL+"?"+params.Encode(), nil)
		if err != nil {
			return result, err
		}
		req.Header.Set("User-Agent", "wiki2docx/1.0 (github.com/w0ikid/wiki2docx)")

		resp, err := wikiClient.Do(req)
		if err != nil {
			return result, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return result, fmt.Errorf("API returned status %d", resp.StatusCode)
		}

		var res struct {
			Query struct {
				Random []struct {
					Title string `json:"title"`
				} `json:"random"`
			} `json:"query"`
		}

		err = json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()
		if err != nil {
			return result, err
		}

		if len(res.Query.Random) == 0 {
			break
		}

		for _, r := range res.Query.Random {
			if _, exists := uniqueTitles[r.Title]; !exists {
				uniqueTitles[r.Title] = struct{}{}
				result = append(result, r.Title)
				if len(result) >= limit {
					break
				}
			}
		}
	}

	return result, nil
}
