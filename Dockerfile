# syntax=docker/dockerfile:1.7

FROM golang:1.26.1-alpine AS builder

WORKDIR /src

RUN apk add --no-cache nodejs npm

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY web/package.json web/package-lock.json ./web/
RUN --mount=type=cache,target=/root/.npm \
    cd web && npm ci --cache /root/.npm --prefer-offline

COPY . .

ARG VERSION=dev

RUN --mount=type=cache,target=/root/.npm \
    PATH="/src/web/node_modules/.bin:$PATH" ./web/node_modules/.bin/buf generate --template buf.gen.yaml
RUN --mount=type=cache,target=/root/.npm \
    cd web && PUBLIC_APP_VERSION=${VERSION} npm run build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go generate ./...
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X github.com/Y4shin/open-caucus/cmd.version=${VERSION}" -o /out/conference-tool .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /workspace

COPY --from=builder /out/conference-tool /usr/local/bin/conference-tool

ENTRYPOINT ["conference-tool"]
CMD ["serve"]
