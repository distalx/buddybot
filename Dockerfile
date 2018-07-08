FROM golang:1.10.3-alpine as builder

# build directories
RUN mkdir /go/src/app
COPY . /go/src/app
WORKDIR /go/src/app

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
COPY --from=builder /go/src/app/buddybot /buddybot
CMD ["/buddybot"]