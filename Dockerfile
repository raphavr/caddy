FROM golang:1.12.3-alpine

RUN apk add --no-cache git

WORKDIR $GOPATH/src/github.com/raphavr/caddy

COPY . .

RUN go get -d -v ./...

RUN go install -v ./...

RUN rm -rf $GOPATH/src

EXPOSE 80
EXPOSE 443

CMD ["caddy", "--conf", "/etc/Caddyfile"]

COPY resource/Caddyfile /etc/Caddyfile