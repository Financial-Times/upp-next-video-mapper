FROM alpine:3.3

ADD *.go /next-video-mapper/

RUN apk add --update bash \
  && apk --update add git bzr \
  && apk --update add go \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/next-video-mapper" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && mv next-video-mapper/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get -t ./... \
  && go build \
  && mv next-video-mapper /next-video-mapper-app \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/*

CMD [ "/next-video-mapper-app" ]