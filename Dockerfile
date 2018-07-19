FROM golang:1.10.3-alpine as builder

# build directories
RUN mkdir -p /go/src/github.com/billglover/buddybot
COPY ./ /go/src/github.com/billglover/buddybot/
WORKDIR /go/src/github.com/billglover/buddybot

# # Go dep!
# Note: can't use alpine base image as it doesn't include git. This also introduces an online dependency.
# RUN go get -u github.com/golang/dep/...
# RUN dep ensure

# Build application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -a -installsuffix cgo -ldflags '-w -s' -o buddybot .

FROM scratch

# Copy certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy application
COPY --from=builder /go/src/github.com/billglover/buddybot/buddybot /buddybot
CMD ["/buddybot"]