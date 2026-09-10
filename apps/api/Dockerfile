FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY apps/api/go.mod apps/api/go.sum* ./
RUN go mod download
COPY apps/api ./
RUN go build -o /out/cal-api ./cmd/server
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.25.0

FROM alpine:3.22
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=build /out/cal-api /usr/local/bin/cal-api
COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY apps/api/migrations ./migrations
COPY apps/api/entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh
EXPOSE 8080
CMD ["./entrypoint.sh"]
