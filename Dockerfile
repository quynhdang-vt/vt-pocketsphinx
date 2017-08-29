## docker build --build-ag
FROM alpine:3.6 as alpine-tools
MAINTAINER Quynh Dang
LABEL alpine-version=3.6 cmu-sphinx=latest

RUN apk update && apk add -U ffmpeg
RUN apk update && apk add -U build-base go git curl libstdc++ alpine-sdk python-dev swig libtool autoconf automake bison file
RUN apk add pulseaudio-dev --update-cache --repository http://dl-3.alpinelinux.org/alpine/edge/testing/ --allow-untrusted

FROM alpine-tools as alpine-cmu-pocketsphinx
# CMU Sphinx
RUN mkdir /cmusphinx && cd /cmusphinx && git clone --recursive https://github.com/cmusphinx/sphinxbase && cd /cmusphinx/sphinxbase && ./autogen.sh && ./configure && make clean all && make check && make install && rm -rf /cmusphinx/sphinxbase

# CMU Pocketsphinx
RUN cd /cmusphinx && git clone --recursive https://github.com/cmusphinx/pocketsphinx.git && cd /cmusphinx/pocketsphinx && ./autogen.sh && ./configure && make clean all && make check && make install && rm -rf /cmusphinx

ENV GOPATH /go
ENV PATH $PATH:/usr/local/go/bin:/go/bin
ENV LD_LIBRARY_PATH $LD_LIBRARY_PATH:/usr/local/lib

FROM alpine-cmu-pocketsphinx as qd-pocketsphinx0
ARG GITHUB_TOKEN

ADD . /go/src/github.com/quynhdang-vt/vt-pocketsphinx
RUN  chmod +x /go/src/github.com/quynhdang-vt/vt-pocketsphinx/setupgo.sh && /go/src/github.com/quynhdang-vt/vt-pocketsphinx/setupgo.sh && mkdir -p /var/log/qd-pocketsphinx && cd /go/src/github.com/quynhdang-vt/vt-pocketsphinx && go get -u github.com/govend/govend && /go/bin/govend -v && go build -o /go/bin/qd-pocketsphinx *.go && rm -f /root/.netrc && rm -rf /go/src/github.com/quynhdang-vt/vt-pocketsphinx
ENTRYPOINT ["/go/bin/qd-pocketsphinx"]

## for Veritone
FROM qd-pocketsphinx0 as qd-pocketsphinx
ADD manifest.json /var/
