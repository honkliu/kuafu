package api

import (
	"os"
	"strings"
	"testing"
)

func TestOpenAPIListsRegisteredRoutes(t *testing.T) {
	data, err := os.ReadFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	spec := string(data)
	for _, route := range []string{
		"/health:",
		"/ready:",
		"/api/v1/cluster/summary:",
		"/api/v1/nodes:",
		"/api/v1/nodes/{name}:",
		"/api/v1/gpus:",
		"/api/v1/gpus/{id}:",
		"/api/v1/jobs:",
		"/api/v1/jobs/{id}:",
		"/api/v1/jobs/{id}/start:",
		"/api/v1/jobs/{id}/stop:",
		"/api/v1/jobs/{id}/restart:",
		"/api/v1/jobs/{id}/logs:",
		"/api/v1/jobs/{id}/metrics:",
		"/api/v1/reservations:",
		"/api/v1/reservations/{id}:",
		"/api/v1/reservations/{id}/commands:",
		"/api/v1/queues:",
		"/api/v1/queues/{name}:",
	} {
		if !strings.Contains(spec, route) {
			t.Fatalf("OpenAPI spec missing route %s", route)
		}
	}
}
