FROM golang:1.16 AS build
ENV GO111MODULE=on

WORKDIR $GOPATH/src/bootjp/vrc_panoprama_picture_manage
COPY . .
RUN go get github.com/rakyll/statik && $GOPATH/bin/statik -src=./public
RUN GOOS=linux OARCH=arm64 CGO_ENABLED=0 go build -a -o out cli/main.go && cp out /app

FROM multiarch/ubuntu-core:arm64-bionic
COPY --from=build /app /app

CMD ["/app"]
