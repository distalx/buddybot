.PHONY: clean build deploy

clean: 
	go clean ./cmd
	rm -rf ./deploy/*
	
build:
	GOOS=linux GOARCH=amd64 go build -o deploy/main ./cmd
	zip -j deploy/cmd.zip deploy/main
	rm -f deploy/main

deploy:
	#aws s3 mb s3://me.billglover.buddybot
	aws s3 cp deploy/cmd.zip s3://me.billglover.buddybot/cmd.zip
	aws lambda update-function-code --function-name buddybot-CommandHandler-1VA6KIISC3W1V --s3-bucket me.billglover.buddybot --s3-key cmd.zip --region eu-west-1

sam:
	aws cloudformation deploy --region eu-west-1 --capabilities CAPABILITY_IAM --template-file /Users/bill/go/src/github.com/billglover/buddybot/aws/deployment.yaml --stack-name buddybot
	aws cloudformation list-exports --region eu-west-1
