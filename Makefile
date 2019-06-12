
.PHONY: format
format:
	goimports -l -w -local github.com/howardjohn/file-based-istio *.go cmd/*.go client/*.go

.PHONY: install
install:
	go install -v

.PHONY: deploy
deploy:
	helm template install | kubectl replace --force -f -

.PHONY: clean
clean:
	rm -r install/files
	mkdir install/files