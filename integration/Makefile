# master makefile

default: all

all: containers

containers:
	cd contest; make

compose/test.yml: tmpl/contest.yml
	mkdir -p compose
	./buildtest.sh $(PASSWD)

starttest: compose/test.yml
	docker-compose -f compose/test.yml up -d

endtest:
	docker-compose -f compose/test.yml stop
	docker-compose -f compose/test.yml rm -f

testdev: containers compose/test.yml starttest

cleanslate: endtest clean

clean:
	rm -rf compose

realclean: clean
	cd contest; make realclean
