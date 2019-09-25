.PHONY: build clean deploy

build:
	dep ensure -v
	env GOOS=linux go build -ldflags="-s -w" -o bin/create_user cmd/user/create_user/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/delete_user cmd/user/delete_user/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/list_user cmd/user/list_user/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/forgot_password cmd/user/forgot_password/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/confirm_forgot_password cmd/user/confirm_forgot_password/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/auto_verify cmd/user/auto_verify/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/verify_email cmd/user/verify_email/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/authenticate_user cmd/user/authenticate_user/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/create_user_pool cmd/userpool/create_userpool/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/list_user_pool cmd/userpool/list_userpool/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/list_user_pool_client cmd/userpool/list_userpool_client/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/describe_user_pool_client cmd/userpool/describe_userpool_client/main.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/create_user_pool_client cmd/userpool/create_userpool_client/main.go

clean:
	rm -rf ./bin ./vendor Gopkg.lock

deploy: clean build
	sls deploy --verbose
