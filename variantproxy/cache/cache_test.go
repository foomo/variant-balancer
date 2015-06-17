package cache

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestGetExpiredItems(t *testing.T) {
	c := NewCache(time.Second, time.Second*300)
	id := "foo"
	uri := "/"
	data := []byte("just a test")
	header := http.Header{}
	c.Save(id, uri, data, header)
	c.Get(id)
	time.Sleep(time.Second * 2)

	expiredAfterTwoSeconds := c.getExpiredItems(time.Second * 1)
	if len(expiredAfterTwoSeconds) < 1 {
		t.Fatal("at least one should have expired")
	}

	expiredAfterFiveSeconds := c.getExpiredItems(time.Second * 5)
	if len(expiredAfterFiveSeconds) > 0 {
		t.Fatal("none should have failed")
	}

}

func TestEviction(t *testing.T) {
	c := NewCache(time.Second, time.Second)
	Debug = true

	id := "foo"
	uri := "/"
	data := []byte("just a test")
	header := http.Header{}

	makeFoo := func(newId string) {
		debug("--------------- making " + newId)
		id = newId
		if c.GetLock(id) {
			debug("-------------------- got lock")
			c.Save(id, uri, data, header)
		} else {
			c.WaitFor(id)
			debug("-------------------- waited for it")
			c.Save(id, uri, data, header)
		}
		debug("-------------------- made "+id, c.cache)
	}

	makeFoo("foo 1")

	assert.Len(t, c.cache, 1)
	time.Sleep(time.Second * 3)
	assert.Len(t, c.cache, 0)

	makeFoo("foo 2")

	if c.Get(id) == nil {
		t.Fatal("failed to get item")
	}
	for i := 0; i < 22; i++ {
		time.Sleep(time.Millisecond * 100)
		assert.NotNil(t, c.Get(id))
	}

	assert.Len(t, c.cache, 1)
	time.Sleep(time.Second * 3)
	assert.Len(t, c.cache, 0)
}

func TestSaveGet(t *testing.T) {
	Debug = true
	c := NewCache(time.Second*5, time.Second*300)
	id := "foo"
	uri := "/"
	data := []byte("just a test")
	header := http.Header{}
	c.Save(id, uri, data, header)
	item := c.Get(id)
	assert.NotNil(t, item)
	if item != nil {
		assert.Equal(t, id, item.Id)
	}

}

func TestLocking(t *testing.T) {
	var wg sync.WaitGroup
	c := NewCache(time.Second, time.Second*300)
	id := "foo"
	assert.True(t, c.GetLock(id))
	wg.Add(1)
	go func() {
		assert.False(t, c.GetLock(id))
		c.WaitFor(id)
		assert.True(t, c.GetLock(id))
		wg.Done()
	}()
	time.Sleep(time.Millisecond * 100)
	c.Cancel(id)
	wg.Wait()
}
