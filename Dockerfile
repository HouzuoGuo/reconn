FROM ubuntu:23.04
MAINTAINER Houzuo Guo <i@hz.gl>

WORKDIR /
ENV DEBIAN_FRONTEND=noninteractive
RUN apt update && apt upgrade -q -y -f -m -o Dpkg::Options::=--force-confold -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-overwrite && apt install -q -y -f -m -o Dpkg::Options::=--force-confold -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-overwrite busybox
COPY reconn /
COPY reconn-webapp/dist/reconn-webapp/ /resource
ENTRYPOINT ["/reconn"]
