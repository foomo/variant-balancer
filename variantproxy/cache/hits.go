package cache

import (
	//	"log"
	"time"
)

func (c *Cache) hitCounterRoutine() {
	for {
		select {
		case id := <-c.hitChannel:

			hits, ok := c.hits[id]
			if !ok {
				hits = []time.Time{}
			}
			newHits := []time.Time{time.Now()}
			lastIndex := len(hits)
			if lastIndex > 0 {
				if lastIndex > 9 {
					lastIndex = 9
				}
				newHits = append(newHits, hits[:lastIndex]...)
			}
			c.hits[id] = newHits
			debug(len(c.hits[id]), "Hits for resource:", id)
		}
	}
}

func (c *Cache) getHitStats(since time.Time) map[string]bool {
	// map containing the relevance per entry
	hits := make(map[string]bool)
	// delete items without any hits
	for id, _ := range c.cache {
		hits[id] = c.itemHasHits(id, since)
	}
	return hits
}

func (c *Cache) itemHasHits(id string, since time.Time) bool {
	debug("itemHasHits: since", since, "for", id)
	hits, ok := c.hits[id]
	if ok {
		debug("	hits for", id)
		for _, hitTime := range hits {
			if since.Unix() <= hitTime.Unix() {
				debug("		hit", since.Unix(), "<", hitTime.Unix())
				return true
			} else {
				//debug("     not a hit", since.Unix(), ">", hitTime.Unix())
			}
		}
	}
	return false
}
