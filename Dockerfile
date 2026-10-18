# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/goki ./cmd/goki

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/goki /usr/local/bin/goki
WORKDIR /wiki
EXPOSE 4242
# Inside a container goki must listen on all interfaces. Set GOKI_AUTH_USERNAME
# and GOKI_AUTH_PASSWORD, or pass --read-only, before exposing it to a network.
ENTRYPOINT ["goki", "--host", "0.0.0.0", "--port", "4242", "--no-open"]
CMD ["/wiki"]
