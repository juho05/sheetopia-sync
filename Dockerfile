# docker buildx build --platform linux/arm64,linux/amd64 --tag ghcr.io/juho05/sheetopia-sync:dev --push .
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
ARG VERSION=dev
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags "-s -w -X github.com/juho05/sheetopia-sync.Version=$VERSION" -o /out/sheetopia-sync ./cmd/server && \
    go build -trimpath -ldflags "-s -w -X github.com/juho05/sheetopia-sync.Version=$VERSION" -o /out/sheetopia-admin ./cmd/admin

FROM alpine:3.24
COPY --from=build /out/ /bin/

VOLUME /data
EXPOSE 8080

ENV DATA_DIR=/data
ENV PORT=8080
CMD [ "sheetopia-sync" ]
