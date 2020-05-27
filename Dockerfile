FROM golang:alpine AS build
RUN apk add --no-cache git

COPY . /go/src/nano-pp

EXPOSE 6379

RUN ls

WORKDIR /go/src/nano-pp

RUN ls

RUN go get github.com/gomodule/redigo/redis
RUN go get github.com/google/uuid
RUN go get github.com/sacOO7/gowebsocket
RUN go get github.com/adjust/rmq

RUN go build -o /go/bin/nano-pp

ENTRYPOINT [ "/go/bin/nano-pp" ]
