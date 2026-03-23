FROM --platform=$BUILDPLATFORM golang:alpine AS build

RUN --mount=type=cache,target=/var/cache/apk \
    ln -s /var/cache/apk /etc/apk/cache && \
    apk add git

WORKDIR /messageCounter
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go mod download

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o messageCounter

FROM alpine

RUN --mount=type=cache,target=/var/cache/apk \
    ln -s /var/cache/apk /etc/apk/cache && \
    apk add ca-certificates

COPY --from=build /messageCounter/messageCounter /usr/bin/
COPY --from=build /messageCounter/fonts /fonts

CMD ["messageCounter"]
