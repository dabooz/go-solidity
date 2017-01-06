# master makefile

default: all

all: containers

containers:
	cd contest; make

compose/test.yml: tmpl/contest.yml
	mkdir -p compose
	cat tmpl/contest.yml >>compose/test.yml

starttest: compose/test.yml
	docker-compose -f compose/test.yml up -d

endtest:
	docker-compose -f compose/test.yml stop
	docker-compose -f compose/test.yml rm -f

runtest: containers compose/test.yml starttest
	docker-compose -f compose/test.yml logs

compilecontracts:
	cd compilecon; make
	mkdir -p compose
	cat tmpl/compilecon.yml >>compose/test.yml
	docker-compose -f compose/test.yml up -d

refreshJSON:
	docker cp compose_compilecon_1:/root/directory.json ../contracts/
	docker cp compose_compilecon_1:/root/agreements.json ../contracts/

cleanslate: endtest clean

clean:
	rm -rf compose

realclean: clean
	cd contest; make realclean || :
	cd compilecon; make realclean || :
