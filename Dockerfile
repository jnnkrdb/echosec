# ---------------------------------------------------------------------------------------------- Golang
# Building the go binary
#FROM golang:1.19 AS operator
#FROM golang:1.23 AS operator
FROM golang:1.20 AS operator
WORKDIR /echosec-src
# copy the code files
COPY pkg/ /echosec-src/pkg
COPY cmd/ /echosec-src/cmd
COPY api/ /echosec-src/api
COPY internal/controller/ /echosec-src/internal/controller

COPY go.mod /echosec-src/go.mod
COPY go.sum /echosec-src/go.sum

# set env vars
ENV CGO_ENABLED=0
ENV GOARCH=amd64
ENV GOOS=linux

# START BUILD
RUN go mod download && go build -o /echosec cmd/main.go

# ---------------------------------------------------------------------------------------------- Final Alpine
FROM alpine:3.19
LABEL org.opencontainers.image.source="https://github.com/jnnkrdb/echosec"
LABEL org.opencontainers.image.description="Operator for cluster-wide mirroring of secrets/configmaps."
WORKDIR /

# install neccessary binaries
RUN apk add --no-cache --update openssl

# Copy the echosec Directory Contents
COPY opt/ /opt

# create vault user with home dir
RUN addgroup -S echosec && adduser -S echosec -H -h /opt/echosec -s /bin/sh -G echosec -u 3453

# Copy Operators Binary
COPY --from=operator /echosec /usr/local/bin/echosec
RUN chmod 700 /usr/local/bin/echosec &&\
    chmod 700 -R /opt/echosec &&\
    chown echosec:echosec /usr/local/bin/echosec &&\
    chown echosec:echosec -R /opt/echosec
    
# change user to echosec user
USER echosec:echosec

# set the entrypoints
ENTRYPOINT ["/opt/echosec/entrypoint.sh"]
CMD [ "echosec" ]