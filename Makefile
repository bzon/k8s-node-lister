build:
	go get ./...
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/node-lister
	docker build -t bzon/node-lister .
push:
	docker push bzon/node-lister:latest
