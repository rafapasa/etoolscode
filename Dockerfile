# eToolsCode — imagem oficial
FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/etoolscode ./cmd/etoolscode

FROM alpine:3.20
RUN apk add --no-cache ca-certificates git bash tzdata \
 && adduser -D -u 1000 etools
WORKDIR /app
COPY --from=builder /out/etoolscode /usr/local/bin/etoolscode
# static/templates embutidos via go:embed no binário
ENV HTTP_PORT=8080
ENV HOME=/data
ENV ISSUES_DIR=/data/issues
# auth off por default no container; defina DSAC_UI_PASSWORD_HASH no compose/k8s
ENV DSAC_UI_AUTH=off
EXPOSE 8080
VOLUME ["/data"]
USER etools
ENTRYPOINT ["etoolscode", "--ui"]
