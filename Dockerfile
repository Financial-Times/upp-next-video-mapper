FROM golang:1.8-alpine

ENV PROJECT=upp-next-video-mapper
COPY . /${PROJECT}-sources/

RUN apk --no-cache --virtual .build-dependencies add git \
  && ORG_PATH="github.com/Financial-Times" \
  && REPO_PATH="${ORG_PATH}/${PROJECT}" \
  && mkdir -p $GOPATH/src/${ORG_PATH} \
  && ln -s /${PROJECT}-sources $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && BUILDINFO_PACKAGE="${REPO_PATH}/vendor/github.com/Financial-Times/service-status-go/buildinfo." \
  && VERSION="version=$(git describe --tag --always 2> /dev/null)" \
  && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
  && REPOSITORY="repository=$(git config --get remote.origin.url)" \
  && REVISION="revision=$(git rev-parse HEAD)" \
  && BUILDER="builder=$(go version)" \
  && LDFLAGS="-X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
  && echo "Build flags: $LDFLAGS" \
  && echo "Fetching dependencies..." \
  && go get -u github.com/kardianos/govendor \
  && $GOPATH/bin/govendor sync \
  && echo "Building app..." \
  && CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags="${LDFLAGS}" -o /${PROJECT} ${REPO_PATH} \
  && apk del .build-dependencies \
  && rm -rf $GOPATH/src $GOPATH/pkg $GOPATH/.cache $GOPATH/bin /${PROJECT}-sources

WORKDIR /
CMD ["/upp-next-video-mapper"]