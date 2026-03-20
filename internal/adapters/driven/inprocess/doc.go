// Package inprocess provides in-process adapters for embedded mode.
//
// When Graphiti is embedded as a library in a host application, these
// adapters replace their network-based equivalents. Instead of HTTP
// webhooks, the deploy target calls a Go function directly. Instead
// of HTTP callbacks, execution status flows through function calls.
//
// Use [NewDeployTarget] to create a deploy adapter that invokes a
// callback function when a workflow is deployed.
package inprocess
