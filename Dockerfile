FROM alpine:3.8
LABEL maintainer "Raul Naveiras <rnaveiras@gmail.com>"

COPY postgres_exporter /bin/postgres_exporter

USER nobody
EXPOSE 9187
ENTRYPOINT [ "/bin/postgres_exporter" ]
