FROM alpine:3.7
MAINTAINER Raul Naveiras <rnaveiras@gmail.com>

COPY postgres_exporter /bin/postgres_exporter

ENTRYPOINT [ "/bin/postgres_exporter" ]
