FROM marcus-base:latest as build

WORKDIR /audio

COPY .. .
ENV CGO_ENABLED=1

RUN /usr/local/go/bin/go mod download

FROM build as fin

RUN /usr/local/go/bin/go build -o /marcus

CMD ["/marcus"]