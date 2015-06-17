// Provides in memory caching with cache eviction
package cache

import (
	"net/http"
	"time"
)

type waiter struct {
	Id          string
	DoneChannel chan int
}

type itemCallback struct {
	Item        *Item
	DoneChannel chan int
}

type Cache struct {
	ExpirationAge         time.Duration
	cache                 map[string]*Item
	lockIdChannel         chan string
	lockIdResponseChannel chan bool
	waitForIdChannel      chan *waiter
	saveChannel           chan *itemCallback
	cancelChannel         chan *itemCallback
	deleteChannel         chan *itemCallback
	hits                  map[string][]time.Time
	// i told you naming is hard
	hitChannel chan string
}

// Toggle debug log output
var Debug = false

func NewCache(expirationAge time.Duration, evictionInterval time.Duration) *Cache {
	c := &Cache{
		ExpirationAge:         expirationAge,
		cache:                 make(map[string]*Item),
		lockIdChannel:         make(chan string),
		lockIdResponseChannel: make(chan bool),
		waitForIdChannel:      make(chan *waiter),
		saveChannel:           make(chan *itemCallback),
		cancelChannel:         make(chan *itemCallback),
		deleteChannel:         make(chan *itemCallback),
		hits:                  make(map[string][]time.Time),
		hitChannel:            make(chan string),
	}
	// managing hits
	go c.hitCounterRoutine()
	// managing channels for locking
	go c.queueRoutine()
	// starting eviction routine
	go c.evictionRoutine(evictionInterval)
	return c
}

func (c *Cache) Save(id string, uri string, data []byte, header http.Header) {
	item := &Item{
		Id:     id,
		Uri:    uri,
		Data:   data,
		Header: header,
	}
	done := make(chan int)
	c.saveChannel <- &itemCallback{
		Item:        item,
		DoneChannel: done,
	}
	<-done
}

func (c *Cache) delete(id string) {
	item := &Item{
		Id: id,
	}
	done := make(chan int)
	c.deleteChannel <- &itemCallback{
		Item:        item,
		DoneChannel: done,
	}
	<-done
}

func (c *Cache) Cancel(id string) {
	item := &Item{
		Id: id,
	}
	done := make(chan int)
	c.cancelChannel <- &itemCallback{
		Item:        item,
		DoneChannel: done,
	}
	<-done
}

func (c *Cache) Get(id string) *Item {
	item, ok := c.cache[id]
	if ok {
		c.hitChannel <- id
	}
	return item
}

func (c *Cache) WaitFor(id string) {
	done := make(chan int)
	c.waitForIdChannel <- &waiter{
		Id:          id,
		DoneChannel: done,
	}
	<-done
}

func (c *Cache) GetLock(id string) bool {
	c.lockIdChannel <- id
	return <-c.lockIdResponseChannel
}
