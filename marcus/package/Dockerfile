FROM marcus-base:latest

WORKDIR /audio

COPY .. .
ENV CGO_ENABLED=1
RUN /usr/local/go/bin/go build -o /marcus

CMD ["/marcus"]