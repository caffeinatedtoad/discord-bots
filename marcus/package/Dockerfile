ARG BASE_IMAGE_TAG=marcus-base-latest
FROM ghcr.io/caffeinatedtoad/discord-bots:${BASE_IMAGE_TAG} AS build

LABEL org.opencontainers.image.source=https://github.com/caffeinatedtoad/discord-bots
LABEL org.opencontainers.image.description="Marcus"

WORKDIR /audio

COPY .. .
ENV CGO_ENABLED=1
ENV PATH="/usr/local/go/bin:${PATH}"

RUN go mod download

FROM build AS fin

RUN go build -o /marcus

CMD ["/marcus"]
