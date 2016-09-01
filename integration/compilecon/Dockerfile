FROM ethereumbuild

RUN apt-get -y install vim

COPY client.sh /root/
COPY odminer.sh /root/
COPY tempssh /root/.ssh/
COPY tempfs /tmp/

ENV GOPATH /tmp
WORKDIR /tmp
RUN mkdir /tmp/bin
RUN mkdir /tmp/pkg

ENV mtn_soliditycontract=1
ENV mtn_soliditycontract_syncdelay=180
ENV mtn_soliditycontract_txdelay=180
ENV mtn_soliditycontract_integration=1

WORKDIR /tmp/src/repo.hovitos.engineering/MTN/go-solidity/clients/device
RUN make
RUN cp /tmp/bin/device /usr/bin/mtn-device_owner

WORKDIR /tmp/src/repo.hovitos.engineering/MTN/go-solidity/clients/bootstrap
RUN make compile
RUN cp /tmp/bin/bootstrap /usr/bin/mtn-bootstrap

RUN rm -fr /tmp/src/repo.hovitos.engineering/MTN/go-solidity/contracts/*.json

WORKDIR /tmp

ENTRYPOINT ["sh", "/root/client.sh"]

