FROM golang:1.24.5-alpine3.21@sha256:6edc20586dd08dacad538c1f09984bc2aa61720be59056cf75429691f294d731 AS builder

ENV PROMU_SHA256=f92fd94dbd5941c7f2925860c3d6a1f24b7630cb2b192df43835c8dda9e76b5d \
  PROMU_VERSION=0.15.0

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

FROM alpine:3.22@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
LABEL org.opencontainers.image.authors="Raul Naveiras <rnaveiras@gmail.com>"

COPY --from=builder /go/src/app/output/postgres_exporter /bin/postgres_exporter

USER nobody
EXPOSE 9187
ENTRYPOINT [ "/bin/postgres_exporter" ]
