package collector

import (
	"context"

	pgx "github.com/jackc/pgx/v4"
	"github.com/prometheus/client_golang/prometheus"

	"time"
)

type dbname string

type cachedScraper struct {
	scraper      Scraper
	ttl          time.Duration
	lastScrapeAt map[dbname]time.Time
	lastValues   map[dbname][]prometheus.Metric
}

func NewCachedScraper(scraper Scraper, ttl time.Duration) Scraper {
	return &cachedScraper{
		scraper:      scraper,
		ttl:          ttl,
		lastScrapeAt: make(map[dbname]time.Time),
		lastValues:   make(map[dbname][]prometheus.Metric),
	}
}
func (c *cachedScraper) Name() string {
	return "Cached" + c.scraper.Name()
}

func (c *cachedScraper) Scrape(ctx context.Context, conn *pgx.Conn, version Version, ch chan<- prometheus.Metric) error {
	key := (dbname)(conn.Config().Database)

	if c.shouldScrape(key) {
		c.lastScrapeAt[key] = time.Now()
		var newValues []prometheus.Metric

		interceptorCh := make(chan prometheus.Metric)
		go func() {
			for {
				v, ok := <-interceptorCh
				if ok {
					ch <- v
					newValues = append(newValues, v)
				} else {
					return
				}
			}
		}()

		err := c.scraper.Scrape(ctx, conn, version, interceptorCh)
		close(interceptorCh)
		c.lastValues[key] = newValues
		return err
	} else {
		values, ok := c.lastValues[key]
		if ok {
			for _, metric := range values {
				ch <- metric
			}
		}
		return nil
	}
}

func (c *cachedScraper) shouldScrape(key dbname) bool {
	lastScrapeAt, ok := c.lastScrapeAt[key]
	if !ok {
		lastScrapeAt = time.Unix(0, 0)
	}

	nextScrapeAt := lastScrapeAt.Add(c.ttl)

	return nextScrapeAt.Before(time.Now())
}
