FROM golang:1.15.3-alpine

WORKDIR /src
# get dependencies
# - use go mod tidy if `go.sum` doesn't exist
COPY ./go.mod ./go.sum ./

# Copy only Go package directories each separately
# then `docker build` does not
COPY cli ./cli/

# See which patch version of Go is used (important for security updates)
RUN go version

RUN CGO_ENABLED=0         \
    GOOS=linux            \
    go install            \
      -a                  \
      --installsuffix cgo \
      --ldflags="-s"      \
      ./cli


# 2nd stage: embed Go binary in damn small Linux distro (== Alpine)

FROM alpine:latest
WORKDIR /app/
RUN apk --no-cache add ca-certificates
# (optional) If you are doing timezone related processing (not UTC)
# This package is more up to date than Go's `time/tzdata`, see https://github.com/golang/go/issues/38017#issuecomment-619631945
RUN apk --no-cache add tzdata

# Copy the binary from the first build stage
COPY --from=0 /go/bin/cli app