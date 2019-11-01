FROM golang:alpine AS build
ENV GO111MODULE=on
ENV REPO_NAME=github.com/bootjp/vrc_panoprama_picture_manage

RUN apk add --no-cache git
RUN \
  cd $GOPATH/src/ && \
  mkdir -p $REPO_NAME && \
  cd github.com/bootjp/ && \
  git clone https://$REPO_NAME.git && \
  cd ./vrc_panoprama_picture_manage && \
  GOOS=linux CGO_ENABLED=0 go build -a -o out cli/main.go && \
  cp out /app

FROM alpine
RUN apk add --no-cache tzdata ca-certificates
COPY ./public/index.html /public/index.html
COPY --from=build /app /app

CMD ["/app"]
