ARG PG_CONTAINER_VERSION=16

FROM postgres:${PG_CONTAINER_VERSION}-bookworm AS builder

ARG DEBIAN_FRONTEND=noninteractive

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
      build-essential \
      git \
      postgresql-server-dev-${PG_MAJOR} \
      pkg-config \
      binutils \
      automake \
      autoconf \
      libtool \
      ca-certificates; \
    rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    git clone --branch 1.2.3 --single-branch --depth 1 https://github.com/hightman/scws.git /tmp/scws; \
    cd /tmp/scws; \
    touch README; \
    aclocal; \
    autoconf; \
    autoheader; \
    libtoolize; \
    automake --add-missing; \
    ./configure --prefix=/usr/local; \
    make; \
    make install

RUN set -eux; \
    git clone --branch master --single-branch --depth 1 https://github.com/amutu/zhparser.git /tmp/zhparser; \
    cd /tmp/zhparser; \
    make install

FROM postgres:${PG_CONTAINER_VERSION}-bookworm

COPY --from=builder /usr/lib/postgresql/${PG_MAJOR}/lib/zhparser.so /usr/lib/postgresql/${PG_MAJOR}/lib/
COPY --from=builder /usr/local/lib/libscws.* /usr/local/lib/
COPY --from=builder /usr/share/postgresql/${PG_MAJOR}/extension/zhparser* /usr/share/postgresql/${PG_MAJOR}/extension/
COPY --from=builder /usr/lib/postgresql/${PG_MAJOR}/lib/bitcode/zhparser* /usr/lib/postgresql/${PG_MAJOR}/lib/bitcode/
COPY --from=builder /usr/share/postgresql/${PG_MAJOR}/tsearch_data/*.utf8.* /usr/share/postgresql/${PG_MAJOR}/tsearch_data/

RUN set -eux; \
    echo "/usr/local/lib" > /etc/ld.so.conf.d/usr-local.conf; \
    ldconfig
