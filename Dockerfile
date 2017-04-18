FROM alpine:3.4

ARG PROJECT=upp-next-video-mapper

COPY . /${PROJECT}/

RUN apk add --update bash \
  && apk --update add git bzr \
  && apk --update add go \
  && export GOPATH=/gopath \
  && REPO_ROOT="github.com/Financial-Times/" \
  && mkdir -p $GOPATH/src/${REPO_ROOT}/ \
  && mv ${PROJECT} $GOPATH/src/${REPO_ROOT} \
  && cd $GOPATH/src/${REPO_ROOT}/${PROJECT} \
  && go get -t ./... \
  && go test \
  && go build \
  && mv ${PROJECT} / \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/*

CMD [ "/upp-next-video-mapper" ]