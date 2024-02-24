FROM golang:1.22

ENV GO111MODULE on

RUN go version

RUN apt-get update

RUN apt-get install clang pkg-config libwayland-dev libx11-dev libx11-xcb-dev libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev libxcursor-dev libvulkan-dev  libbz2-dev libmpc-dev libmpfr-dev libgmp-dev liblzma-dev libpsi3-dev libssl-dev libxml2-dev libz-dev lzma-dev icoutils --yes

RUN go install gioui.org/cmd/gogio@latest
#Define the working directory in the container
WORKDIR /app
