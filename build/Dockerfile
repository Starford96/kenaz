FROM alpine:latest

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/* && mkdir /app

COPY build/app /app/app

WORKDIR /app

CMD ["/app/app"]
