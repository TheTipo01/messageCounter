FROM --platform=$BUILDPLATFORM golang:alpine AS build

RUN apk add --no-cache git

RUN git clone https://github.com/TheTipo01/messageCounter /messageCounter
WORKDIR /messageCounter
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go mod download
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o messageCounter

FROM alpine

COPY --from=build /messageCounter/messageCounter /usr/bin/
COPY --from=build /messageCounter/fonts /fonts

CMD ["messageCounter"]
