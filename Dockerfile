FROM golang:1

ENV PROJECT=upp-next-video-mapper
ENV ORG_PATH="github.com/Financial-Times"
ENV DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)"
ENV SRC_FOLDER="${GOPATH}/src/${ORG_PATH}/${PROJECT}"

COPY . ${SRC_FOLDER}
WORKDIR ${SRC_FOLDER}

ARG GITHUB_USERNAME
ARG GITHUB_TOKEN

# Install dependancies and build app
RUN BUILDINFO_PACKAGE="${ORG_PATH}/service-status-go/buildinfo." \
    && VERSION="version=$(git describe --tag --always 2> /dev/null)" \
    && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
    && REPOSITORY="repository=$(git config --get remote.origin.url)" \
    && REVISION="revision=$(git rev-parse HEAD)" \
    && BUILDER="builder=$(go version)" \
    && GOPRIVATE="github.com/Financial-Times" \
    && git config --global url."https://${GITHUB_USERNAME}:${GITHUB_TOKEN}@github.com".insteadOf "https://github.com" \
    && LDFLAGS="-X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
    && CGO_ENABLED=0 go build -mod=readonly -a -o /artifacts/${PROJECT} -ldflags="${LDFLAGS}" \
    && echo "Build flags: ${LDFLAGS}"

# Multi-stage build - copy certs and the binary into the image
FROM scratch
WORKDIR /
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /artifacts/* /

CMD [ "/upp-next-video-mapper" ]
