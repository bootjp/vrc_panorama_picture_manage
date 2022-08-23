FROM golang:1.19 AS build

WORKDIR $GOPATH/src/bootjp/vrc_panoprama_picture_manage
COPY . .
RUN go install github.com/rakyll/statik@latest && statik -src=./public
RUN GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -a -o out cli/main.go && cp out /app

FROM gcr.io/distroless/static:latest-arm64
COPY --from=build /app /app

CMD ["/app"]
