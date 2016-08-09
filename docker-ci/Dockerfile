FROM ubuntu:14.04
MAINTAINER Spruce Backend <backend@sprucehealth.com>
RUN apt-get update # January 20, 2015
RUN apt-get -y upgrade
RUN DEBIAN_FRONTEND=noninteractive apt-get -y install mysql-client-5.6 mysql-server-5.6 postgresql-9.3 \
	ruby unzip zip curl libelf1 git python-pip python-magic python-dateutil tmux memcached && \
	apt-get clean
RUN pip install --upgrade https://github.com/s3tools/s3cmd/releases/download/v1.6.1/s3cmd-1.6.1.tar.gz
RUN gem install sass

# Install Flow
RUN cd /usr/local && \
	curl -s -o flow.zip https://carefront-packages.s3.amazonaws.com/flow-linux64-v0.18.1.zip && \
	unzip flow.zip && \
	ln -s /usr/local/flow/flow /usr/local/bin/flow && \
	rm flow.zip

# Install Go
RUN cd /usr/local && \
	rm -rf go && \
	curl -s https://storage.googleapis.com/golang/go1.7rc6.linux-amd64.tar.gz | tar -zxf - && \
	ln -s /usr/local/go/bin/go /usr/local/bin/go

# Install Consul
RUN curl -s -o /usr/local/bin/consul.zip https://releases.hashicorp.com/consul/0.6.3/consul_0.6.3_linux_amd64.zip && \
	cd /usr/local/bin && \
	unzip consul.zip && \
	rm consul.zip

# Install node.js + npm
RUN cd /usr/local && \
	curl -s http://carefront-packages.s3.amazonaws.com/node-v5.2.0-linux-x64.tar.gz | tar -zxf - && \
	ln -s /usr/local/node-v5.2.0-linux-x64/bin/node /usr/local/bin/node && \
	ln -s /usr/local/node-v5.2.0-linux-x64/bin/npm /usr/local/bin/npm

# Install docker
RUN cd /tmp && \
	curl -s https://get.docker.com/builds/Linux/x86_64/docker-latest.tgz | tar -zxf - && \
	chmod +x /tmp/docker/* && \
	mv /tmp/docker/* /usr/bin/ && \
	rm -rf /tmp/docker

COPY ./my.cnf /etc/mysql/my.cnf
COPY ./start.sh /start.sh
COPY ./run.sh /run.sh
CMD ["/bin/bash", "-x", "/start.sh"]
