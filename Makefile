DOCKER_IMAGE_NAME = billglover/buddybot

default:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o buddybot .

build:
	dep ensure
	docker build -t billglover/buddybot .

run:
	docker run --env BUDDYBOT_TOKEN billglover/buddybot

clean:
	go clean