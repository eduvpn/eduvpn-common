.PHONY: compile test

compile:
	python3 -m discovery

test:
	$(MAKE) -C ../../exports
	python3 -m unittest test_discovery
