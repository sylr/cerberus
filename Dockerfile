ARG CERBERUS_IMAGE_VERSION=dev

# ------------------------------------------------------------------------------

FROM cerberus-go:$CERBERUS_IMAGE_VERSION as go

# ------------------------------------------------------------------------------

FROM debian:10-slim

RUN apt-get update && apt-get dist-upgrade -y

COPY --from=go /go/src/github.com/sylr/cerberus/dist/cerberus /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/cerberus", "-a", "0.0.0.0:80"]
