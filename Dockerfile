FROM --platform=linux/amd64 jrei/systemd-debian:13

RUN apt-get update
RUN apt-get install -y openjdk-25-jre-headless curl
RUN apt-get clean
RUN rm -rf /var/lib/apt/lists/*

COPY mcsd /usr/local/bin/mcsd

WORKDIR /srv