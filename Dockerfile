FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache make

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN make build BUILD_FLAGS="-a" LDFLAGS="-s -w"

FROM alpine:latest

LABEL io.maxstash.image.source="https://github.com/maxmorhardt/olympics-api"
LABEL io.maxstash.image.description="Olympics API - team generation, group stage, and single-elimination brackets"
LABEL io.maxstash.image.vendor="Max Morhardt"

ENV GIN_MODE="release"

WORKDIR /app

RUN addgroup -g 1000 olympics && \
    adduser -D -u 1000 -G olympics olympics

COPY --from=build --chown=olympics:olympics /src/bin/olympics-api .

RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates && \
    chmod +x olympics-api

USER olympics

EXPOSE 8080

CMD ["./olympics-api"]
