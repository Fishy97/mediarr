FROM node:24-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
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
COPY --from=backend /out/media-steward /app/media-steward
COPY --from=frontend /app/frontend/dist /app/web
ENV MEDIA_STEWARD_ADDR=:8080
ENV MEDIA_STEWARD_CONFIG_DIR=/config
ENV MEDIA_STEWARD_FRONTEND_DIR=/app/web
EXPOSE 8080
VOLUME ["/config"]
ENTRYPOINT ["/app/media-steward"]

