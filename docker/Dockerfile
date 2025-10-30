FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api ./cmd/api

FROM alpine:3.20
WORKDIR /app
COPY --from=build /bin/api /app/api
COPY --from=build /src/internal/storage/migrations /app/internal/storage/migrations
COPY config /app/config
EXPOSE 8080
ENTRYPOINT ["/app/api"]
