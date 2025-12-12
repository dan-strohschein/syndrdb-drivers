package client

// Version is set by build flags during compilation.
// Example: go build -ldflags "-X github.com/dan-strohschein/syndrdb-drivers/src/golang/client.Version=$(git describe --tags --always --dirty)"
var Version = "dev"
