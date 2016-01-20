FROM ethereum

COPY client.sh /root/
COPY tempssh /root/.ssh/

WORKDIR /tmp
RUN mkdir /tmp/src
RUN mkdir /tmp/bin
RUN mkdir /tmp/pkg
RUN mkdir /tmp/src/repo.hovitos.engineering
RUN mkdir /tmp/src/repo.hovitos.engineering/MTN
WORKDIR /tmp/src/repo.hovitos.engineering/MTN
RUN git clone ssh://git@repo.hovitos.engineering:10022/MTN/go-solidity.git
WORKDIR /tmp/src/repo.hovitos.engineering/MTN/go-solidity
RUN git checkout indefinite
WORKDIR /tmp/src/repo.hovitos.engineering/MTN/go-solidity/clients
ENV GOPATH /tmp
ENV mtn_soliditycontract=1
ENV mtn_soliditycontract_txdelay=120
RUN make
RUN cp /tmp/bin/device /usr/bin/mtn-device_owner
RUN cp /tmp/bin/rest_container_provider /usr/bin/mtn-rest_container_provider
RUN cp /tmp/bin/bootstrap /usr/bin/mtn-bootstrap
RUN cp /tmp/bin/directory_test /usr/bin/mtn-directory_test

WORKDIR /tmp/src
RUN go get github.com/gorilla/mux
RUN git clone ssh://git@repo.hovitos.engineering:10022/MTN-research/marketplace.git
WORKDIR /tmp/src/marketplace
RUN git checkout indefinite
WORKDIR /tmp/src/marketplace/gorest
RUN make
RUN cp /tmp/bin/gorest /usr/bin/mtn-gorest

ENTRYPOINT ["sh", "/root/client.sh"]

