SHELL := /bin/bash

clean:
	echo "nothing to clean"
run-simple: clean
	go run simplevariantbalancer.go
test:
	go test -v  -timeout="20s" github.com/foomo/variant-balancer/variantproxy github.com/foomo/variant-balancer/variantbalancer github.com/foomo/variant-balancer/usersessions
	
