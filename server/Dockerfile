### STAGE 1: Build server component

FROM golang:alpine as go-builder
RUN apk update && apk upgrade && apk add --no-cache bash git openssh gcc

ENV CGO_ENABLED=0

RUN mkdir /build 
ADD . /build/
WORKDIR /build 
ENV GIT_TERMINAL_PROMPT=1
RUN go build -o main .

### STAGE 2: Final container ###

FROM golang:alpine

LABEL maintainer Igor Rendulic <igor@chryscloud.com>

RUN apk --no-cache --update add ca-certificates

COPY --from=go-builder /build/main /app/

ENTRYPOINT [ "/app/main" ]