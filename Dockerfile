FROM golang:1.25.5-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/conference-tool .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /workspace

COPY --from=builder /out/conference-tool /usr/local/bin/conference-tool

ENTRYPOINT ["conference-tool"]
CMD ["serve"]
