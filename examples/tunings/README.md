# Running the example:

## Set up the environment variables:

### For VertexAI Backend

```
export GOOGLE_GENAI_USE_VERTEXAI=true
export GOOGLE_CLOUD_PROJECT={YOUR_PROJECT_ID}
export GOOGLE_CLOUD_LOCATION={YOUR_LOCATION}
```

### For GeminiAPI Backend
```
export GOOGLE_GENAI_USE_VERTEXAI=false
export GOOGLE_API_KEY={YOUR_API_KEY}
```

Once you setup the environment variables, you can download, build and run the
example using the following commands.

```
$ go get google.golang.org/genai
$ cd `go list -f '{{.Dir}}' google.golang.org/genai/examples/tunings`
$ go run list.go
$ go run create_get.go

// You can also specify --model arg for create_get.go
// go run create_get.go --model=gemini-2.5-flash
```