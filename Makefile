PKGS=./...

default: vet test

vet:
	go vet ${PKGS}

test:
	go test ${PKGS}

test-race:
	go test -race ${PKGS}

bench:
	go test ${PKGS} -run=NONE -bench=. -benchmem

bench-race:
	go test ${PKGS} -run=NONE -bench=. -benchmem -race

.PHONY: default vet test
