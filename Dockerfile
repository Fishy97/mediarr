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
RUN CGO_ENABLED=0 go build -o /out/mediarr ./cmd/mediarr
RUN CGO_ENABLED=0 go build -o /out/mediarr-acceptance ./cmd/mediarr-acceptance

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg tzdata
WORKDIR /app
RUN addgroup -S mediarr && adduser -S -G mediarr -u 10001 mediarr \
  && mkdir -p /config \
  && chown -R mediarr:mediarr /app /config
COPY --from=backend /out/mediarr /app/mediarr
COPY --from=backend /out/mediarr-acceptance /app/mediarr-acceptance
COPY --from=frontend /app/frontend/dist /app/web
RUN chown -R mediarr:mediarr /app
ENV MEDIARR_ADDR=:8080
ENV MEDIARR_CONFIG_DIR=/config
ENV MEDIARR_FRONTEND_DIR=/app/web
EXPOSE 8080
VOLUME ["/config"]
USER mediarr
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 CMD wget -qO- http://127.0.0.1:8080/api/v1/health >/dev/null || exit 1
ENTRYPOINT ["/app/mediarr"]
