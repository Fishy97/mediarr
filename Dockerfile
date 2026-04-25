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
RUN CGO_ENABLED=0 go build -o /out/media-steward ./cmd/medi-steward

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg tzdata
WORKDIR /app
RUN addgroup -S media-steward && adduser -S -G media-steward -u 10001 media-steward \
  && mkdir -p /config \
  && chown -R media-steward:media-steward /app /config
COPY --from=backend /out/media-steward /app/media-steward
COPY --from=frontend /app/frontend/dist /app/web
RUN chown -R media-steward:media-steward /app
ENV MEDIA_STEWARD_ADDR=:8080
ENV MEDIA_STEWARD_CONFIG_DIR=/config
ENV MEDIA_STEWARD_FRONTEND_DIR=/app/web
EXPOSE 8080
VOLUME ["/config"]
USER media-steward
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 CMD wget -qO- http://127.0.0.1:8080/api/v1/health >/dev/null || exit 1
ENTRYPOINT ["/app/media-steward"]
