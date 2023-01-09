.PHONY: all build test

test:
	@PROJECT_ROOT=$(PWD) APP_ENV=test go test -timeout 5s -tags grpc,pulsar,newrelic,kitex $(shell go list ./plugins/... | grep -v mocks)
