.PHONY: build run clean docker docker-run

build:
	go build -o leadgen-mcp .

run: build
	./leadgen-mcp -config config.yaml

clean:
	rm -f leadgen-mcp leadgen-mcp.exe

docker:
	docker build -t leadgen-mcp:latest .

docker-run: docker
	docker run --rm -p 8080:8080 -v $(PWD)/config.yaml:/app/config.yaml leadgen-mcp:latest
