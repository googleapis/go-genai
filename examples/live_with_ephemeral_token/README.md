# Running the example:

## Set up the environment variables:

### For GeminiAPI Backend
```
export GOOGLE_GENAI_USE_VERTEXAI=false
export GOOGLE_API_KEY={YOUR_API_KEY}
```

Once you setup the environment variables, you can download, build and run the
example using the following commands.

```
$ go get google.golang.org/genai
$ cd `go list -f '{{.Dir}}' google.golang.org/genai/examples/live_with_ephemeral_token`
$ go run *.go
```
