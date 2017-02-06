FROM ethereumbuild

RUN apt-get update
RUN apt-get -y install vim

COPY client.sh /root/
COPY tempssh /root/.ssh/
COPY tempfs /tmp/

ENV GOPATH /tmp
WORKDIR /tmp
RUN mkdir /tmp/bin
RUN mkdir /tmp/pkg

ENV mtn_soliditycontract=1
ENV mtn_soliditycontract_syncdelay=180
ENV mtn_soliditycontract_txdelay=60
ENV mtn_soliditycontract_integration=1
ENV mtn_soliditycontract_block_read_delay=1
ENV mtn_soliditycontract_logstats=1

WORKDIR /tmp/src/github.com/open-horizon/go-solidity/clients
RUN make
RUN cp /tmp/bin/bootstrapv2 /usr/bin/mtn-bootstrapv2
RUN cp /tmp/bin/directory_test /usr/bin/mtn-directory_test
RUN cp /tmp/bin/agreement_protocol /usr/bin/mtn-agreement_protocol_test

WORKDIR /tmp

ENTRYPOINT ["sh", "/root/client.sh"]

