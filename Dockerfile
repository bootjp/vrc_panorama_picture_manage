FROM golang:latest AS build

WORKDIR $GOPATH/src/bootjp/vrc_panoprama_picture_manage
COPY . .
RUN go install github.com/rakyll/statik@latest && statik -src=./public
RUN GOOS=linux CGO_ENABLED=0 go build -a -o out cli/main.go && cp out /app
RUN wget http://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz
# COPY ffmpeg-release-amd64-static.tar.xz ffmpeg-release-amd64-static.tar.xz

RUN apt-get -y update
RUN apt-get install -y xz-utils liblzma-dev
RUN tar Jxfv ./ffmpeg-release-amd64-static.tar.xz
RUN cp ./ffmpeg-*-amd64-static/ffmpeg /tmp/ffmpeg

FROM gcr.io/distroless/static:latest
COPY --from=build /app /app
COPY --from=build /tmp/ffmpeg /usr/bin/ffmpeg

CMD ["/app"]
