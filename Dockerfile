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

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /api

## Deploy
FROM python:3.9-alpine

RUN mkdir -p /home/developer
COPY --from=build /api /home/developer/api
COPY PyDofus /home/developer/PyDofus

RUN apk add sudo

RUN export uid=1000 gid=1000 && \
    echo "developer:x:${uid}:${gid}:Developer,,,:/home/developer:/bin/bash" >> /etc/passwd && \
    echo "developer:x:${uid}:" >> /etc/group && \
    echo "developer ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/developer && \
    chmod 0440 /etc/sudoers.d/developer && \
    chown ${uid}:${gid} -R /home/developer

WORKDIR /home/developer

USER developer

EXPOSE 3000

CMD [ "/home/developer/api" ]