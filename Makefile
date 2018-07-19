DOCKER_IMAGE_NAME = billglover/buddybot

default:
	CGO_ENABLED=0 go build -a -installsuffix cgo -o buddybot .

build:
	#dep ensure
	docker build -t billglover/buddybot .

run:
	docker run --env BUDDYBOT_TOKEN --env BUDDYBOT_SIGNING_SECRET --env BUDDYBOT_PORT -p=3000:3000 billglover/buddybot

clean:
	go clean