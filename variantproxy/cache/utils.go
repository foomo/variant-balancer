package cache

import (
	"log"
)

func debug(a ...interface{}) {
	if Debug {
		log.Println(a...)
	}
}
