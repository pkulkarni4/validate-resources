FROM golang:alpine as builder

RUN apk update && apk add git && apk add ca-certificates

#ADD ./notary /notary
#RUN chmod -R 777 /notary

#WORKDIR /plugin

#RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ./image/admission-webhook ./plugin/admission-webhook

COPY ./admission-webhook /

ENTRYPOINT ["/admission-webhook"]