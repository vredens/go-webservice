.PHONY: test build

test:
	go test -v ./...

test-race:
	go test -v -race ./...

bench:
	go test -v -tags bench -bench Bench ./...

coverage:
	go test -v -coverprofile=.coverage.out -timeout 30s ./...
	go tool cover -func=.coverage.out

%: 
	@echo "not a valid target"
	@:
