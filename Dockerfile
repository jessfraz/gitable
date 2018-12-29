FROM golang:alpine as builder
MAINTAINER Jessica Frazelle <jess@linux.com>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN	apk add --no-cache \
	bash \
	ca-certificates

COPY . /go/src/github.com/jessfraz/gitable

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		git \
		gcc \
		libc-dev \
		libgcc \
		make \
	&& cd /go/src/github.com/jessfraz/gitable \
	&& make static \
	&& mv gitable /usr/bin/gitable \
	&& apk del .build-deps \
	&& rm -rf /go \
	&& echo "Build complete."

FROM alpine:latest

COPY --from=builder /usr/bin/gitable /usr/bin/gitable
COPY --from=builder /etc/ssl/certs/ /etc/ssl/certs

ENTRYPOINT [ "gitable" ]
CMD [ "--help" ]
