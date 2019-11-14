all: btest
	@btest/btest

btest:
	@curl -s https://www.zeek.org/downloads/btest-0.59.tar.gz \
		| tar -xzf -
	@mv btest-0.59 btest

clean:
	@rm -f .btest.failed.dat
	@rm -rf .tmp
