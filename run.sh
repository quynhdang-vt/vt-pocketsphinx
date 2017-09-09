#!/bin/bash
IMG_NAME=qd-pocketsphinx
VT_DOCKER_REG=docker.aws-dev.veritone.com/14667
VT_IMG_TAG=v3
VT_IMG_NAME=${VT_DOCKER_REG}/${IMG_NAME}:${VT_IMG_TAG}
ENGINE_ID=0fdfa1bc-1248-4e0a-a58b-013356343a23

if [ $# -lt 1 ];
then
  opt="build"
else
  opt=$1
fi

echo OPTION=$opt

if [ $opt == 'help' ];
then
  echo "$0 {help, run, push} to run or push.  No option means building."
elif [ $opt == 'run' ];
then
    docker run -it --entrypoint=sh -v /Users/home/go/src/github.com/quynhdang-vt/vt-pocketsphinx:/go/src/github.com/quynhdang-vt/vt-pocketsphinx -v /Users/home/Documents/testdata:/testdata -e GHT=${GITHUB_TOKEN} -e PAYLOAD_FILE=/testdata/vt/engine/payload.json ${IMG_NAME}
elif [ $opt == 'push' ];
then
    start=`date +%s`
    docker push ${VT_IMG_NAME} 
    end=`date +%s`
    runtime=$((end-start))
    echo "PUSHING took $runtime sec"
elif [ $opt == 'build' ];
then
    #set ENGINE_ID in manifest.json and Dockerfile
    sed "s/ENGINE_ID/${ENGINE_ID}/g" manifest.json.template > manifest.json
    docker build --squash -t ${IMG_NAME} --build-arg GHT=${GITHUB_TOKEN} --build-arg ENGINE_ID=${ENGINE_ID} .
    docker tag ${IMG_NAME} ${VT_IMG_NAME}
fi
