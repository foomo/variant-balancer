package cache

import (
	"time"
)

func (c *Cache) evictionRoutine(interval time.Duration) {
	for {
		// there are items in the cache
		debug("Cache Eviction started, currently", len(c.cache), "items in cache.")
		debug("===================================================================")
		expiredItems := c.getExpiredItems(c.ExpirationAge)
		debug("Items picked for eviction", len(expiredItems))
		debug("===================================================================")
		c.deleteItems(expiredItems, 50*time.Millisecond)
		debug("===================================================================")
		debug("deleted cache items, currently", len(c.cache), "items in cache.")
		debug("===================================================================")
		debug("Cache Eviction sleeping for", interval)
		time.Sleep(interval)
	}
}

func (c *Cache) getExpiredItems(maxAge time.Duration) []string {
	allHits := c.getHitStats(getExpirationTime(maxAge))
	expiredItems := []string{}
	for id, hasHits := range allHits {
		if hasHits == false {
			expiredItems = append(expiredItems, id)
		}
	}
	return expiredItems
}

func getExpirationTime(maxAge time.Duration) time.Time {
	secondsSince := time.Now().Unix() - maxAge.Nanoseconds()/1e9
	return time.Unix(secondsSince, 0)
}

// Delete a map of Ressource IDs in the background, with a given interval
func (c *Cache) deleteItems(items []string, pause time.Duration) {
	for _, id := range items {
		if c.GetLock(id) {
			if false == c.itemHasHits(id, getExpirationTime(c.ExpirationAge)) {
				debug("deleting item", id)
				c.delete(id)
			}
			time.Sleep(pause)
		}
	}
}
