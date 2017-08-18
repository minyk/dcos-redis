FROM ubuntu:16.04
RUN apt-get update && apt-get install -y \
    git \
    curl \
    jq \
    default-jdk \
    python-pip \
    python3 \
    python3-dev \
    python3-pip \
    tox \
    software-properties-common \
    python-software-properties \
    libssl-dev \
    wget \
    zip && \
    # install go 1.7
    add-apt-repository -y ppa:longsleep/golang-backports && \
    apt-get update && apt-get install -y golang-go && \
    apt-get clean
# AWS CLI for uploading build artifacts
RUN pip install awscli
# shakedown and dcos-cli require this to output cleanly
ENV LC_ALL=C.UTF-8
ENV LANG=C.UTF-8
# use an arbitrary path for temporary build artifacts
ENV GOPATH=/go-tmp