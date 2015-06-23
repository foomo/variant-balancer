SHELL := /bin/bash

options:
	echo "i do noting yet"
clean:
run-simple:
	make clean
	godebug run -instrument=github.com/foomo/variant-balancer/variantproxy simplevariantbalancer.go
run-simple:
	make clean
	go run simplevariantbalancer.go
test:
	go test -v  -timeout="20s" github.com/foomo/variant-balancer/variantproxy/cache github.com/foomo/variant-balancer/variantproxy github.com/foomo/variant-balancer/usersessions
