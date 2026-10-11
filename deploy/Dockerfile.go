FROM golang:1.27.2-bookworm AS build
WORKDIR /src
COPY go-prototype/go.mod go-prototype/go.sum ./
RUN go mod download
COPY go-prototype/ ./
RUN GOMAXPROCS=2 CGO_ENABLED=0 go build -p 2 -trimpath -o /weatherdesk ./cmd/weatherdesk
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/* && useradd -u 10001 -m desk && mkdir /data && chown desk:desk /data
COPY --from=build /weatherdesk /usr/local/bin/weatherdesk
USER desk
ENV WEATHER_DESK_ROOT=/data WEATHER_DESK_HTTP_ADDR=0.0.0.0:8087
ENTRYPOINT ["weatherdesk"]
