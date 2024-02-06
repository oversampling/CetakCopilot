FROM golang:1.21.6-alpine3.18 as dev

WORKDIR /code

RUN go install github.com/cosmtrek/air@latest