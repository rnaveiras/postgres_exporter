FROM golang:1.16-alpine3.12 as builder

ENV PROMU_SHA256=17d991fea8e992636272d9b4e24b72c104d305ed1460d40891894fae6b6c64a2 \
    PROMU_VERSION=0.12.0

RUN set -x \
  && apk --no-cache add curl ca-certificates git \
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

FROM alpine:3.12
LABEL maintainer="Raul Naveiras <rnaveiras@gmail.com>"

COPY --from=builder /go/src/app/output/postgres_exporter /bin/postgres_exporter

USER nobody
EXPOSE 9187
ENTRYPOINT [ "/bin/postgres_exporter" ]
