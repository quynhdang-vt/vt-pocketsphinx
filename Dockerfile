FROM alpine:3.5

ARG GITHUB_TOKEN

ADD . /go/src/github.com/veritone/task-cmu-sphinx-containerized

# apk add --update alpine-sdk
RUN apk update && \
    apk add -U build-base go git curl libstdc++ alpine-sdk vim tree python python-dev swig libtool autoconf automake bison && \
		cd /go/src/github.com/veritone/task-cmu-sphinx-containerized && \
    git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"
RUN apk add pulseaudio-dev --update-cache --repository http://dl-3.alpinelinux.org/alpine/edge/testing/ --allow-untrusted

RUN mkdir /cmusphinx && cd /cmusphinx && git clone --recursive https://github.com/cmusphinx/sphinxbase
RUN cd /cmusphinx/sphinxbase && ./autogen.sh && ./configure && make clean all && make check && make install

RUN cd /cmusphinx && git clone --recursive https://github.com/cmusphinx/pocketsphinx.git
RUN cd /cmusphinx/pocketsphinx && ./autogen.sh && ./configure && make clean all && make check && make install

RUN cd /usr/local && curl -O https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz && tar -xvf go1.8.linux-amd64.tar.gz && rm -f go1.8.linux-amd64.tar.gz
ENV GOPATH /go
ENV PATH $PATH:/usr/local/go/bin
ENV LD_LIBRARY_PATH $LD_LIBRARY_PATH:/usr/local/lib

ADD .netrc /root/
RUN go get github.com/jawher/mow.cli && go get github.com/xlab/closer && go get github.com/xlab/pocketsphinx-go/sphinx && go get github.com/xlab/pocketsphinx-go/pocketsphinx
ADD ffmpeg/ffmpeg /veritone/
ADD manifest.json /var/

VOLUME /veritone/testdata
ADD . /go/src/github.com/quynhdang-vt/qd-pocketsphinx
RUN  cd /go/src/github.com/quynhdang-vt/qd-pocketsphinx && go get -u github.com/govend/govend && govend -v && go build main.go && mv main /usr/local/go/bin/qd-pocketphinx && rm -rf /go/src/github.com/quynhdang-vt

ADD testdata /veritone/testdata
