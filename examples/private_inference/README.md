# Running the example:

## Set up the environment variables:

### For VertexAI Backend

```
export GOOGLE_GENAI_USE_VERTEXAI=true
export GOOGLE_CLOUD_PROJECT={YOUR_PROJECT_ID}
export GOOGLE_CLOUD_LOCATION={YOUR_LOCATION}
```

Once you setup the environment variables, update the `model` variable and
 `caPool` identifier in `main.go` with your private inference configuration:

```go
	model := "{YOUR_PRIVATE_INFERENCE_MODEL_ID}"
	caPool := "{YOUR_CA_POOL_PATH}"
```

Then you can download, build, and run the example. To leverage BoringSSL, you should explicitly compile and run using the `GOEXPERIMENT=boringcrypto` environment flag:

```
$ go get google.golang.org/genai
$ cd `go list -f '{{.Dir}}' google.golang.org/genai/examples/private_inference`
$ GOEXPERIMENT=boringcrypto go run basic.go
```

