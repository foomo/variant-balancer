package cache

func (c *Cache) queueRoutine() {
	locks := make(map[string][]*waiter)

	notifyWaiters := func(id string) {
		waiters, ok := locks[id]
		if ok {
			delete(locks, id)
			go func() {
				for _, waiter := range waiters {
					waiter.DoneChannel <- 0
				}
			}()
		}
	}

	for {
		select {
		case lockId := <-c.lockIdChannel:
			_, locked := locks[lockId]
			if locked {
				// sbdy else got the lock
				c.lockIdResponseChannel <- false
			} else {
				// you won it is yours
				locks[lockId] = []*waiter{}
				c.lockIdResponseChannel <- true
			}
		case deleter := <-c.deleteChannel:
			notifyWaiters(deleter.Item.Id)
			delete(c.cache, deleter.Item.Id)
			delete(c.hits, deleter.Item.Id)
			deleter.DoneChannel <- 0
		case canceller := <-c.cancelChannel:
			notifyWaiters(canceller.Item.Id)
			canceller.DoneChannel <- 0
		case waiter := <-c.waitForIdChannel:
			locks[waiter.Id] = append(locks[waiter.Id], waiter)
		case saver := <-c.saveChannel:
			c.cache[saver.Item.Id] = saver.Item
			c.hitChannel <- saver.Item.Id
			notifyWaiters(saver.Item.Id)
			saver.DoneChannel <- 0
		}
	}
}
