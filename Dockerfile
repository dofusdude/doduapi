# syntax=docker/dockerfile:1

FROM golang:1.19-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY gen ./gen
COPY server ./server
COPY utils ./utils
COPY update ./update
COPY main.go ./

#RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /api
RUN GOOS=linux GOARCH=amd64 go build -o /api

## Deploy
FROM python:3.9-alpine

WORKDIR /app

COPY --from=build /api /app/api

COPY PyDofus /app/PyDofus

EXPOSE 3000

CMD [ "/app/api" ]