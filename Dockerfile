# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom
ARG GO_VERSION
ARG NODE_VERSION

FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS web
WORKDIR /workspace/web
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN npm install --global "$(node --print 'require("./package.json").packageManager')"
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG GO_LICENSES_VERSION
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/google/go-licenses/v2@${GO_LICENSES_VERSION}
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/embed.go
COPY --from=web /workspace/web/dist web/dist
COPY .mise/tasks/notices .mise/tasks/notices
RUN ./.mise/tasks/notices
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags "-s -w -X main.version=${VERSION#v}" -o /sigweft ./cmd/sigweft

FROM gcr.io/distroless/static:nonroot
COPY LICENSE /LICENSE
COPY --from=builder /workspace/third_party_licenses /third_party_licenses
COPY --from=builder /usr/local/go/LICENSE /third_party_licenses/go/LICENSE
COPY --from=builder /sigweft /sigweft
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/sigweft"]
CMD ["serve"]
