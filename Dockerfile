FROM golang:alpine3.11 AS builder
MAINTAINER lucmichalski <luc.michalski@protonmail.com>

COPY . /go/src/github.com/lucmichalski/peaks-seeker
WORKDIR /go/src/github.com/lucmichalski/peaks-seeker

RUN go install

FROM alpine:3.11 AS runtime
MAINTAINER lucmichalski <luc.michalski@protonmail.com>

ARG TINI_VERSION=${TINI_VERSION:-"v0.18.0"}

# Install tini to /usr/local/sbin
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-muslc-amd64 /usr/local/sbin/tini

# Install runtime dependencies & create runtime user
RUN apk --no-cache --no-progress add ca-certificates \
	&& chmod +x /usr/local/sbin/tini && mkdir -p /opt \
	&& adduser -D lucmichalski -h /opt/peaks-seeker -s /bin/sh \
	&& su lucmichalski -c 'cd /opt/peaks-seeker; mkdir -p bin config data ui'

# Switch to user context
USER lucmichalski
WORKDIR /opt/lucmichalski/data

# copy executable
COPY --from=builder /go/bin/peaks-seeker /opt/lucmichalski/bin/peaks-seeker

ENV PATH $PATH:/opt/lucmichalski/bin

# Container configuration
VOLUME ["/opt/lucmichalski/data"]
ENTRYPOINT ["tini", "-g", "--"]
CMD ["/opt/lucmichalski/bin/peaks-seeker"]

