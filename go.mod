module google.golang.org/genai/v2

go 1.24

// v1.11.0 was retracted (a breaking change on GenerateVideos). The public
// module is major version 2 (the export appends /v2 to the module path), and
// a v2 module cannot retract v1 versions;
// the retraction stays in v1's go.mod.

require (
	cloud.google.com/go v0.116.0
	cloud.google.com/go/auth v0.9.3
	github.com/eliben/go-sentencepiece v0.7.0
	github.com/google/go-cmp v0.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/stretchr/testify v1.8.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.66.2 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

