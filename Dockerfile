FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY . .
ARG VERSION=0.7.0
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" -o /out/goinception-plus ./cmd/goinception-plus

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/goinception-plus /goinception-plus
COPY config/config.minimal.toml /etc/goinception-plus/config.toml
EXPOSE 4000 4001
ENTRYPOINT ["/goinception-plus"]
CMD ["-config", "/etc/goinception-plus/config.toml"]
