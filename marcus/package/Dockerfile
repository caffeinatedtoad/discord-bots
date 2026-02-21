ARG BASE_IMAGE_TAG=marcus-base-latest
FROM ghcr.io/caffeinatedtoad/discord-bots:${BASE_IMAGE_TAG} as build

LABEL org.opencontainers.image.source=https://github.com/caffeinatedtoad/discord-bots
LABEL org.opencontainers.image.description="Marcus"

WORKDIR /audio

COPY .. .
ENV CGO_ENABLED=1

RUN /usr/local/go/bin/go mod download

FROM build as fin

RUN /usr/local/go/bin/go build -o /marcus

CMD ["/marcus"]