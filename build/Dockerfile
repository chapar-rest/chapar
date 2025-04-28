FROM ubuntu:latest

RUN apt-get update && apt-get install -y \
    build-essential \
    dh-make \
    devscripts \
    lintian \
    golang

WORKDIR /app
COPY . .
