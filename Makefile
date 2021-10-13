go-%:
	$(MAKE) -f go.mk $*

.PHONY: clean build-and-docker docker

clean: go-clean

build-and-docker: go-build-for-docker docker

docker:
	docker build -t subshellgmbh/o-neko-url-trigger .
