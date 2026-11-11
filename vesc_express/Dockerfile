ARG BUILD_FROM

# Wird vom Supervisor nativ auf dem HA-Gerät gebaut (passende Architektur)
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY app/go.mod app/go.sum ./
RUN go mod download
COPY app/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /vesc-express .

FROM $BUILD_FROM
COPY --from=build /vesc-express /usr/bin/vesc-express
# s6-overlay umgehen: /init leert sonst die Umgebung (SUPERVISOR_TOKEN fehlt)
ENTRYPOINT ["/usr/bin/vesc-express"]
CMD []
