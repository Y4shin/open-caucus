# syntax=docker/dockerfile:1.7

FROM golang:1.25.5-alpine AS builder

WORKDIR /src

RUN apk add --no-cache nodejs npm

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --cache /root/.npm --prefer-offline

COPY web/package.json web/package-lock.json ./web/
RUN --mount=type=cache,target=/root/.npm \
    cd web && npm ci --cache /root/.npm --prefer-offline

COPY . .

RUN --mount=type=cache,target=/root/.npm \
    npm run build:css
RUN --mount=type=cache,target=/root/.npm \
    cd web && npx buf generate --template ../buf.gen.yaml
RUN --mount=type=cache,target=/root/.npm \
    cd web && npm run build
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go generate ./...
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/conference-tool .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /workspace

COPY --from=builder /out/conference-tool /usr/local/bin/conference-tool

ENTRYPOINT ["conference-tool"]
CMD ["serve"]
