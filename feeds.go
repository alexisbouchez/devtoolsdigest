package main

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

type FeedSource struct {
	Name     string `json:"name"`
	Feed     string `json:"feed"`
	Category string `json:"category"`
}

type Article struct {
	Title     string
	Link      string
	Source    string
	Category string
	Published time.Time
}

func loadFeedSources(path string) ([]FeedSource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sources []FeedSource
	if err := json.Unmarshal(data, &sources); err != nil {
		return nil, err
	}
	return sources, nil
}

func fetchAllArticles(sources []FeedSource, since time.Duration) []Article {
	var (
		mu       sync.Mutex
		articles []Article
		wg       sync.WaitGroup
	)

	cutoff := time.Now().Add(-since)
	parser := gofeed.NewParser()

	for _, src := range sources {
		wg.Add(1)
		go func(s FeedSource) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			feed, err := parser.ParseURLWithContext(s.Feed, ctx)
			if err != nil {
				return
			}

			for _, item := range feed.Items {
				pub := itemPublished(item)
				if pub.IsZero() || pub.Before(cutoff) {
					continue
				}
				mu.Lock()
				articles = append(articles, Article{
					Title:     item.Title,
					Link:      item.Link,
					Source:    s.Name,
					Category: s.Category,
					Published: pub,
				})
				mu.Unlock()
			}
		}(src)
	}

	wg.Wait()

	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Published.After(articles[j].Published)
	})

	return articles
}

func itemPublished(item *gofeed.Item) time.Time {
	if item.PublishedParsed != nil {
		return *item.PublishedParsed
	}
	if item.UpdatedParsed != nil {
		return *item.UpdatedParsed
	}
	return time.Time{}
}
