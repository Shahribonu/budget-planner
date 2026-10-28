# One image: the Go server, which also serves the built Mini App.

FROM node:22-alpine AS app
WORKDIR /src/app
COPY app/package.json app/package-lock.json ./
RUN npm ci
COPY app/ ./
RUN npm run build

FROM golang:1.26-alpine AS server
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /out/server /server
COPY --from=app /src/app/dist /app
ENV STATIC_DIR=/app PORT=8080
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/server"]
