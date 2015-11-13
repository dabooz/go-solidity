# Makefile for ethereum

VER = v1
IMAGE = contest

default: $(IMAGE)-$(VER).tar

$(IMAGE)-$(VER).tar: Dockerfile client.sh
	mkdir -p tempssh
	cp /root/.ssh/* ./tempssh/
	docker build --no-cache -t $(IMAGE):$(VER) .
	rm -fr tempssh
	docker tag -f $(IMAGE):$(VER) $(IMAGE):latest
	#docker save -o $(IMAGE)-$(VER).tar $(IMAGE):$(VER)
	touch $(IMAGE)-$(VER).tar

clean:
	rm -rf $(IMAGE)-$(VER).tar
	rm -fr tempssh

dockerclean:
	docker rmi $(IMAGE):latest $(IMAGE):$(VER)

realclean:
	rm -rf $(IMAGE)-$(VER).tar
	rm -fr tempssh
	docker rmi $(IMAGE):latest $(IMAGE):$(VER) || true

