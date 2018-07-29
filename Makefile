.PHONY: clean build deploy

clean:
	rm -rf ./deploy
	rm -rf ./tmp

build: clean
	@mkdir -p ./deploy
	@mkdir -p ./tmp

	@echo
	@echo "Build command handler function:"
	GOOS=linux GOARCH=amd64 go build -o tmp/main ./cmd
	zip -j deploy/cmd.zip ./tmp/main
	rm -f tmp/main

	@echo
	@echo "Build action handler function:"
	GOOS=linux GOARCH=amd64 go build -o tmp/main ./action
	zip -j deploy/action.zip ./tmp/main
	rm -f tmp/main

	@echo
	@echo "Build event handler function:"
	GOOS=linux GOARCH=amd64 go build -o tmp/main ./event
	zip -j deploy/event.zip ./tmp/main
	rm -f tmp/main

	rm -rf ./tmp
	@echo
	@echo "Build artifacts:"
	@ls -ogh deploy/*
