package cache

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/base64"
	"log"
)

func createHashFromUri(uri string) string {
	twentyBytes := sha1.Sum([]byte(uri))
	bytes := []byte{}
	return base64.URLEncoding.EncodeToString(append(bytes, twentyBytes[0:20]...))
}

func compress(data []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write(data)
	w.Close()
	return b.Bytes()
}

func debug(a ...interface{}) {
	if Debug {
		log.Println(a...)
	}
}
