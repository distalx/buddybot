S3_BUCKET := me.billglover.buddybot
SAM_TEMPLATE := $(shell pwd)/deploy/sam.yaml

.PHONY: clean build package deploy

test:
	go test -v ./...

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

clean:
	rm -rf ./deploy
	rm -rf ./tmp

package: build
	aws cloudformation package --template-file sam.yaml --s3-bucket $(S3_BUCKET) --output-template-file $(SAM_TEMPLATE)

stage: package
	aws cloudformation deploy --capabilities CAPABILITY_IAM --template-file $(SAM_TEMPLATE) --stack-name "BuddyBot-DEV" --parameter-overrides EnvName=DEV
	aws cloudformation describe-stacks --stack-name BuddyBot-DEV --query 'Stacks[0].Outputs[*].{Fn:OutputKey,URL:OutputValue}' --output=text

prod: package
	aws cloudformation deploy --capabilities CAPABILITY_IAM --template-file $(SAM_TEMPLATE) --stack-name "BuddyBot-PRD" --parameter-overrides EnvName=PRD
	aws cloudformation describe-stacks --stack-name BuddyBot-PRD --query 'Stacks[0].Outputs[*].{Fn:OutputKey,URL:OutputValue}' --output=text
