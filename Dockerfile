FROM golang:1.16 AS build
ENV GO111MODULE=on

WORKDIR $GOPATH/src/bootjp/vrc_panoprama_picture_manage
COPY . .
RUN go get github.com/rakyll/statik && $GOPATH/bin/statik -src=./public
RUN GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -o out cli/main.go && cp out /app

FROM gcr.io/distroless/static@sha256:c63dfa9945cfb5f260c5e73e2d2b4c4ea3f444b00fc45eab3d6bf45d4b4aa122
COPY --from=build /app /app

CMD ["/app"]
