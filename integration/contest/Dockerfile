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

WORKDIR /tmp/src/repo.hovitos.engineering/MTN

RUN git clone ssh://git@repo.hovitos.engineering:10022/MTN/smartcontract-monitor.git
RUN git clone ssh://git@repo.hovitos.engineering:10022/MTN/go-eth-events.git
RUN git clone ssh://git@repo.hovitos.engineering:10022/MTN/go-eth-rpc.git
RUN go get github.com/golang/glog

WORKDIR /tmp/src/repo.hovitos.engineering/MTN/smartcontract-monitor/docker/fs/etc
RUN cp -Rfa ./monitor /etc/

WORKDIR /tmp/src/repo.hovitos.engineering/MTN/smartcontract-monitor
RUN go build && cp ./smartcontract-monitor /usr/bin/smartcontract-monitor

ENV mtn_soliditycontract=1
ENV mtn_soliditycontract_syncdelay=180
ENV mtn_soliditycontract_txdelay=180
ENV mtn_soliditycontract_integration=1

WORKDIR /tmp/src/repo.hovitos.engineering/MTN/go-solidity/clients
RUN make
RUN cp /tmp/bin/device /usr/bin/mtn-device_owner
RUN cp /tmp/bin/rest_container_provider /usr/bin/mtn-rest_container_provider
RUN cp /tmp/bin/bootstrap /usr/bin/mtn-bootstrap
RUN cp /tmp/bin/directory_test /usr/bin/mtn-directory_test
RUN cp /tmp/bin/deregister /usr/bin/mtn-deregister

WORKDIR /tmp/src
RUN go get github.com/gorilla/mux

WORKDIR /tmp/src/marketplace/gorest
RUN make
RUN cp /tmp/bin/gorest /usr/bin/mtn-gorest

WORKDIR /tmp

ENTRYPOINT ["sh", "/root/client.sh"]

