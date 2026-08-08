FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN go mod tidy && go build -o /app/server ./cmd/api

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /app/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
