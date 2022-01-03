name = socksproxy
.PHONY: all test lint $(name) clean

all: test lint $(name)

test:
	go test -cover ./...

lint:
	golangci-lint run -E gofmt -E revive --exclude-use-default=false

$(name):
	go build -o $(name) .

clean:
	rm $(name)
