package wiki

import (
	gowiki "github.com/trietmn/go-wiki"
)

// Article holds the fetched data for a Wikipedia article.
type Article struct {
	Title   string
	Content string
}

// FetchArticle retrieves the full text of a Wikipedia article by title.
func FetchArticle(title string) (Article, error) {
	page, err := gowiki.GetPage(title, -1, false, true)
	if err != nil {
		return Article{}, err
	}

	content, err := page.GetContent()
	if err != nil {
		return Article{}, err
	}

	return Article{
		Title:   page.Title,
		Content: content,
	}, nil
}

// GetRandomTitles returns up to limit random Wikipedia article titles.
// go-wiki caps at 10 per call, so we make multiple calls if needed.
func GetRandomTitles(limit int) ([]string, error) {
	const maxPerCall = 10
	var titles []string

	for len(titles) < limit {
		batch := limit - len(titles)
		if batch > maxPerCall {
			batch = maxPerCall
		}
		results, err := gowiki.GetRandom(batch)
		if err != nil {
			return titles, err
		}
		titles = append(titles, results...)
	}

	return titles, nil
}
