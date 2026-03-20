// Package driving defines the use case interfaces that Graphiti's HTTP
// handlers and WebSocket hub call into.
//
// These are the "left side" of the hexagonal architecture: they represent
// what the outside world can ask Graphiti to do. In embedded mode, the host
// application can also call these directly via graphiti.App.Services.
package driving
