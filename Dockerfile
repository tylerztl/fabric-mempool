FROM golang:1.14.2 AS build

MAINTAINER tailinzhang1993@gmail.com

ENV APP_DIR /go/src/fabric-mempool
RUN mkdir -p $APP_DIR
WORKDIR $APP_DIR
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fabric-mempool main.go

# Create a minimized Docker mirror
FROM scratch AS prod

COPY --from=build /go/src/fabric-mempool/fabric-mempool /fabric-mempool
EXPOSE 8080
