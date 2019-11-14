all: btest
	@btest/btest

btest:
	@mkdir -p build
	@curl -s https://www.zeek.org/downloads/btest-0.59.tar.gz \
	  | tar -xzf - -s /^btest-0.59/btest/

clean:
	@rm -f .btest.failed.dat
	@rm -rf .tmp
