# Running the example:

GenerateVideos is only supported on the Vertex AI backend.

## Set up the environment variables:

```
export GOOGLE_GENAI_USE_VERTEXAI=true
export GOOGLE_CLOUD_PROJECT={YOUR_PROJECT_ID}
export GOOGLE_CLOUD_LOCATION={YOUR_LOCATION}
```

Once you setup the environment variables, you can download, build and run the
example using the following commands.

```
$ go get google.golang.org/genai
$ cd `go list -f '{{.Dir}}' google.golang.org/genai/examples/models/generate_videos`
$ go run videos.go
```
