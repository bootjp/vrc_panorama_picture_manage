FROM golang:alpine AS build
ENV GO111MODULE=on

WORKDIR $GOPATH/src/bootjp/vrc_panoprama_picture_manage
COPY . .
RUN GOOS=linux CGO_ENABLED=0 go build -a -o out cli/main.go && cp out /app

FROM alpine
RUN apk add --no-cache tzdata ca-certificates
COPY --from=build /app /app

CMD ["/app"]
