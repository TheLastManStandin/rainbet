FROM golang:1.22-bookworm AS build

WORKDIR /src

RUN apt-get update \
  && apt-get install -y --no-install-recommends gcc libc6-dev \
  && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /out/rainbet-api ./cmd/api

FROM debian:bookworm-slim

RUN apt-get update \
  && apt-get install -y --no-install-recommends sqlite3 \
  && rm -rf /var/lib/apt/lists/* \
  && useradd --create-home --uid 10001 app \
  && mkdir /data \
  && chown app:app /data

COPY --from=build /out/rainbet-api /usr/local/bin/rainbet-api

USER app

ENV PORT=8080
ENV DATABASE_PATH=/data/rainbet.db

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/rainbet-api"]
