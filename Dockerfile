# syntax=docker/dockerfile:1
FROM golang:1.17.7-alpine3.15 as builder

ENV PROMU_SHA256=41bdeadd6bb761058adc82e1c0fc9951ca3eac9d0556d29bdf01993f7afd1f57 \
    PROMU_VERSION=0.13.0

SHELL ["/bin/ash", "-euox", "pipefail", "-c"]

# hadolint ignore=DL3018
RUN apk --no-cache add curl ca-certificates git \
  && curl -o /tmp/promu.tar.gz -fsL https://github.com/prometheus/promu/releases/download/v${PROMU_VERSION}/promu-${PROMU_VERSION}.linux-amd64.tar.gz \
  && echo "${PROMU_SHA256}  /tmp/promu.tar.gz" | sha256sum -c \
  && tar xvfz /tmp/promu.tar.gz -C /tmp \
  && cp "/tmp/promu-${PROMU_VERSION}.linux-amd64/promu" /bin/promu \
  && chmod +x /bin/promu \
  && rm -fr /tmp/promu*

WORKDIR /go/src/app
COPY . .

RUN set -x \
  && promu build --verbose --prefix=./output \
  && find ./output

FROM alpine:3.15
LABEL maintainer="Raul Naveiras <rnaveiras@gmail.com>"

COPY --from=builder /go/src/app/output/postgres_exporter /bin/postgres_exporter

USER nobody
EXPOSE 9187
ENTRYPOINT [ "/bin/postgres_exporter" ]
