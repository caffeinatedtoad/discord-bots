ARG BASE_IMAGE_TAG=marcus-base-test
FROM ghcr.io/caffeinatedtoad/discord-bots:${BASE_IMAGE_TAG} AS build

LABEL org.opencontainers.image.source=https://github.com/caffeinatedtoad/discord-bots
LABEL org.opencontainers.image.description="Marcus"

# Download libdave
ENV LIB_DAVE_INSTALL_SCRIPT="https://raw.githubusercontent.com/disgoorg/godave/4ef3e359a3af065fd01c58c70c2e4c56dbc5fc42/scripts/libdave_install.sh"

RUN wget $LIB_DAVE_INSTALL_SCRIPT -O install.sh

RUN chmod +x install.sh
RUN SHELL=bash ./install.sh v1.1.1

ENV PKG_CONFIG_PATH="/root/.local/lib/pkgconfig:$PKG_CONFIG_PATH"
ENV PATH="${PATH}:/root/.local/path"

WORKDIR /audio

COPY .. .
ENV CGO_ENABLED=1
ENV PATH="/usr/local/go/bin:${PATH}"

RUN go mod download

FROM build AS fin

RUN go build -o /marcus

CMD ["/marcus"]
