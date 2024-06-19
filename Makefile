run:build
	@./temp/lb
build:
	@go build -o temp/lb ./cmd/main.go
