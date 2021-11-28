.PHONY: build pack test clean

build:
	mvn compile

pack:
	mvn package

test:
	$(MAKE) -C ../../exports
	mvn test

clean:
	mvn clean
