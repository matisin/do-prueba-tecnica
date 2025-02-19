FROM golang:1.23.3-alpine3.19 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN apk --no-cache add git
RUN go mod download
COPY cmd/ ./cmd
COPY internal/ ./internal
COPY pkgs/ ./pkgs

RUN mkdir ./tmp

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o http ./cmd/http

FROM alpine:3.19 AS final
LABEL author="matisin" maintainer="matias.medina.s2@gmail.com"

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=America/Santiago \
    ENV=prod

WORKDIR /app
COPY --from=builder /app/http .

ARG HTTP_PORT=8080
ENV HTTP_PORT=${HTTP_PORT}
EXPOSE ${HTTP_PORT}

ENTRYPOINT ["./http", "--format=json"]
