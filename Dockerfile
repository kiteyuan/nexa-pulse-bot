FROM node:22-alpine AS web
WORKDIR /src
COPY web/admin/package.json web/admin/package-lock.json* web/admin/
COPY web/public/package.json web/public/package-lock.json* web/public/
RUN cd web/admin && npm install && cd ../public && npm install
COPY web/admin web/admin
COPY web/public web/public
COPY internal/webui internal/webui
RUN cd web/admin && npm run build && cd ../public && npm run build

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/webui /src/internal/webui
RUN CGO_ENABLED=0 go build -o /nexa ./cmd/nexa

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /nexa /app/nexa
ENV NEXA_PUBLIC_ADDR=:8080
ENV NEXA_ADMIN_ADDR=:8081
ENV NEXA_SESSION_DIR=/app/data/sessions
EXPOSE 8080 8081
ENTRYPOINT ["/app/nexa"]
