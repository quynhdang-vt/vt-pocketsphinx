## docker build --build-ag
FROM alpine:3.6 as alpine-tools
MAINTAINER Quynh Dang
LABEL alpine-version=3.5 cmu-sphinx=latest

RUN apk update && apk add -U ffmpeg
RUN apk update && apk add -U build-base git curl libstdc++ alpine-sdk vim tree python python-dev swig libtool autoconf automake bison file
RUN apk add pulseaudio-dev --update-cache --repository http://dl-3.alpinelinux.org/alpine/edge/testing/ --allow-untrusted

FROM alpine-tools as alpine-cmu-pocketsphinx
# CMU Sphinx
RUN mkdir /cmusphinx && cd /cmusphinx && git clone --recursive https://github.com/cmusphinx/sphinxbase
RUN cd /cmusphinx/sphinxbase && ./autogen.sh && ./configure && make clean all && make check && make install

# CMU Pocketsphinx
RUN cd /cmusphinx && git clone --recursive https://github.com/cmusphinx/pocketsphinx.git
RUN cd /cmusphinx/pocketsphinx && ./autogen.sh && ./configure && make clean all && make check && make install

# GO 1.8
#RUN cd /usr/local && curl -O https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz && tar -xvf go1.8.linux-amd64.tar.gz && rm -f go1.8.linux-amd64.tar.gz
ENV GOPATH /go
ENV PATH $PATH:/usr/local/go/bin
ENV LD_LIBRARY_PATH $LD_LIBRARY_PATH:/usr/local/lib

FROM alpine-cmu-pocketsphinx as qd-pocketsphinx0
ARG GITHUB_TOKEN
# GO DEP NEEDED?
#RUN go get github.com/jawher/mow.cli && go get github.com/xlab/closer && go get github.com/xlab/pocketsphinx-go/sphinx && go get github.com/xlab/pocketsphinx-go/pocketsphinx

ADD . /go/src/github.com/quynhdang-vt/vt-pocketsphinx
RUN  chmod +x /go/src/github.com/quynhdang-vt/vt-pocketsphinx/setupgo.sh && /go/src/github.com/quynhdang-vt/vt-pocketsphinx/setupgo.sh && mkdir -p /var/log/qd-pocketsphinx 
RUN cd /go/src/github.com/quynhdang-vt/vt-pocketsphinx && go get -u github.com/govend/govend && /go/bin/govend -v &&go build -o qd-pocketsphinx main.go && rm -f /root/.netrc

## for Veritone
FROM qd-pocketsphinx0 as qd-pocketsphinx
ADD manifest.json /var/

ENTRYPOINT ["/go/src/github.com/quynhdang-vt/vt-pocketsphinx/qd-pocketsphinx"]
