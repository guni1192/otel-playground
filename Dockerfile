# syntax=docker/dockerfile:1

ARG GO_VERSION=1.22.1
FROM golang:${GO_VERSION} AS build
WORKDIR /src

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 go build -o /bin/server ./cmd/server

FROM gcr.io/distroless/static-debian12 AS final

ARG UID=1002
USER nonroot
COPY --from=build /bin/server /bin/
EXPOSE 8080

ENTRYPOINT [ "/bin/server" ]
