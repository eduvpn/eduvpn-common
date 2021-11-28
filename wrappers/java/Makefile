.PHONY: build pack test clean

build:
	mvn --no-transfer-progress compile

pack:
	mvn --no-transfer-progress package

test:
	$(MAKE) -C ../../exports
	mvn --no-transfer-progress test

clean:
	mvn --no-transfer-progress clean
