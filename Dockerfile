# Golang backend environment build container
FROM golang:1.17.1-alpine AS build

ENV DEBIAN_FRONTEND noninteractive
MAINTAINER Alireza Josheghani "josheghani.dev@gmail.com"

WORKDIR /build
ADD . .

RUN go build -o bin/fortress-csi .

# Deploy container
FROM alpine AS deploy

ENV DEBIAN_FRONTEND noninteractive
MAINTAINER Alireza Josheghani "josheghani.dev@gmail.com"

RUN mkdir /csi

# Copying environment grpc server binary to /usr/src/app
COPY --from=build /build/bin/fortress-csi /usr/bin/fortress-csi

# Start supervisord
ENTRYPOINT ["/usr/bin/fortress-csi"]
CMD ["--endpoint", "unix:///csi/csi.sock"]

