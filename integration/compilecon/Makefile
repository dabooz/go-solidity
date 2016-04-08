# Makefile for ethereum

VER = v1
IMAGE = compilecon

default: $(IMAGE)-$(VER).tar

$(IMAGE)-$(VER).tar: Dockerfile client.sh
	mkdir -p tempssh
	cp ../docker/fs/root/.ssh/* ./tempssh/
	mkdir -p tempfs/src/repo.hovitos.engineering/MTN/go-solidity
	rsync -avzq --exclude 'integration/' --exclude '.git/' ../../../go-solidity ./tempfs/src/repo.hovitos.engineering/MTN/
	cp ~/dag/full-R23-0000000000000000 ./tempfs/ || :
	docker build --no-cache -t $(IMAGE):$(VER) .
	rm -fr tempssh
	rm -fr tempfs
	docker tag -f $(IMAGE):$(VER) $(IMAGE):latest
	#docker save -o $(IMAGE)-$(VER).tar $(IMAGE):$(VER)
	touch $(IMAGE)-$(VER).tar

clean:
	rm -rf $(IMAGE)-$(VER).tar
	rm -fr tempssh
	rm -fr tempfs

dockerclean:
	docker rmi $(IMAGE):latest $(IMAGE):$(VER)

realclean: clean dockerclean

