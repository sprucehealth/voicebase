FROM ubuntu:latest
MAINTAINER Spruce Backend <backend@sprucehealth.com>
RUN apt-get update # June 2, 2015
RUN apt-get -y upgrade
RUN DEBIAN_FRONTEND=noninteractive apt-get -y install mysql-client-5.6 mysql-server-5.6 postgresql-9.3 \
	ruby npm nodejs unzip zip curl libelf1 git python-pip python-magic python-dateutil tmux
RUN pip install --upgrade https://github.com/s3tools/s3cmd/releases/download/v1.5.2/s3cmd-1.5.2.tar.gz
RUN gem install sass
ADD http://flowtype.org/downloads/flow-linux64-latest.zip /usr/local/flow.zip
RUN cd /usr/local ; unzip flow.zip ; ln -s /usr/local/flow/flow /usr/local/bin/flow
RUN cd /usr/local && \
	rm -rf go && \
	curl -s https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz | tar -zxf - && \
	ln -s /usr/local/go/bin/go /usr/local/bin/go
RUN ln -s /usr/bin/nodejs /usr/local/bin/node
ADD https://dl.bintray.com/mitchellh/consul/0.5.2_linux_amd64.zip /usr/local/bin/
RUN cd /usr/local/bin ; unzip 0.5.2_linux_amd64.zip
COPY ./my.cnf /etc/mysql/my.cnf
COPY ./start.sh /start.sh
COPY ./run.sh /run.sh
CMD ["/bin/bash", "-x", "/start.sh"]
