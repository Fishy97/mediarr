FROM node:24-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend ./
RUN npm run build

FROM golang:1.23-alpine AS backend
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend ./
RUN CGO_ENABLED=0 go build -o /out/mediaar ./cmd/mediaar

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg tzdata
WORKDIR /app
RUN addgroup -S mediaar && adduser -S -G mediaar -u 10001 mediaar \
  && mkdir -p /config \
  && chown -R mediaar:mediaar /app /config
COPY --from=backend /out/mediaar /app/mediaar
COPY --from=frontend /app/frontend/dist /app/web
RUN chown -R mediaar:mediaar /app
ENV MEDIAAR_ADDR=:8080
ENV MEDIAAR_CONFIG_DIR=/config
ENV MEDIAAR_FRONTEND_DIR=/app/web
EXPOSE 8080
VOLUME ["/config"]
USER mediaar
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 CMD wget -qO- http://127.0.0.1:8080/api/v1/health >/dev/null || exit 1
ENTRYPOINT ["/app/mediaar"]
