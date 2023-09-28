# this is a very basic skeleton produced for documentation purposes for now

test:
	# this is all just standard go
	# we probably want to containerize this for reproducibility
	go build ./...
	go test -v -race ./...

lint:
	# we want to lint both code and templates.
	golangci-lint run
	# will want to recursively validate all templates in apps/
	sam validate --lint --template apps/filedropper/template.yaml

package:
	mkdir -p build/
	# build the lambda
	# requires Go to be installed
	sam build --template apps/filedropper/template.yaml
	# requires AWS credentials.
	# currently dynamically generates bucket. We will want to use a fixed set of buckets for our production artifacts.
	sam package --template apps/filedropper/template.yaml --output-template-file build/filedropper.yaml --region us-east-1
