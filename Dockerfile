
FROM golang:1.22.5-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/etoolcode-ia ./cmd/ui
FROM alpine:3.20
RUN apk add --no-cache git bash ca-certificates
WORKDIR /app
COPY --from=builder /app/etoolcode-ia /app/etoolcode-ia
COPY cmd/ui/static /app/cmd/ui/static
COPY cmd/ui/templates /app/cmd/ui/templates
EXPOSE 8080
ENV HTTP_PORT=8080
ENV DSAC_UI_PRO=1
CMD ["/app/etoolcode-ia"]
