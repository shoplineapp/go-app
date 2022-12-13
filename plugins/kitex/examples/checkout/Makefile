init:
	@kitex -type protobuf -module checkout_service --protobuf Mcheckout.proto=checkout_service/kitex_gen checkout.proto

run:
	@go run -tags kitex,newrelic main.go

loadtest:
	@ghz --insecure --proto ./checkout.proto --call CheckoutService.Healthcheck 127.0.0.1:8888 -c 1000 -n 1000000
