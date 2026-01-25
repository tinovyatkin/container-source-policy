package dhi

import (
	"errors"
	"testing"

	"github.com/containers/image/v5/docker/reference"
)

func TestCanMapToDHI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Should map - official Docker Hub images
		{"short name", "alpine:3.18", true},
		{"short name no tag", "alpine", true},
		{"explicit library", "docker.io/library/alpine:3.18", true},
		{"node image", "node:20", true},
		{"golang image", "golang:1.21-alpine", true},

		// Should NOT map - non-library images
		{"docker.io org image", "docker.io/myorg/myimage:1.0", false},
		{"docker.io user image", "docker.io/someuser/app:latest", false},

		// Should NOT map - other registries
		{"ghcr.io", "ghcr.io/actions/runner:latest", false},
		{"quay.io", "quay.io/centos/centos:8", false},
		{"gcr.io", "gcr.io/distroless/static:nonroot", false},
		{"ecr", "123456789.dkr.ecr.us-east-1.amazonaws.com/myapp:v1", false},

		// Should NOT map - dhi.io itself (avoid double mapping)
		{"dhi.io image", "dhi.io/alpine:3.18", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := reference.ParseNormalizedNamed(tt.input)
			if err != nil {
				t.Fatalf("failed to parse reference %q: %v", tt.input, err)
			}

			got := CanMapToDHI(ref)
			if got != tt.expected {
				t.Errorf("CanMapToDHI(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMapToDHI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"alpine with tag", "alpine:3.18", "dhi.io/alpine:3.18"},
		{"alpine no tag", "alpine", "dhi.io/alpine"},
		{"explicit library", "docker.io/library/alpine:3.18", "dhi.io/alpine:3.18"},
		{"node image", "node:20", "dhi.io/node:20"},
		{"golang alpine", "golang:1.21-alpine", "dhi.io/golang:1.21-alpine"},
		{"python slim", "python:3.12-slim", "dhi.io/python:3.12-slim"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := reference.ParseNormalizedNamed(tt.input)
			if err != nil {
				t.Fatalf("failed to parse reference %q: %v", tt.input, err)
			}

			got, err := MapToDHI(ref)
			if err != nil {
				t.Fatalf("MapToDHI(%q) returned error: %v", tt.input, err)
			}
			if got == nil {
				t.Fatalf("MapToDHI(%q) returned nil", tt.input)
			}

			gotStr := got.String()
			if gotStr != tt.expected {
				t.Errorf("MapToDHI(%q) = %q, want %q", tt.input, gotStr, tt.expected)
			}
		})
	}
}

func TestMapToDHI_NotEligible(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"ghcr.io", "ghcr.io/actions/runner:latest"},
		{"docker.io org", "docker.io/myorg/myimage:1.0"},
		{"quay.io", "quay.io/centos/centos:8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := reference.ParseNormalizedNamed(tt.input)
			if err != nil {
				t.Fatalf("failed to parse reference %q: %v", tt.input, err)
			}

			got, err := MapToDHI(ref)
			if !errors.Is(err, ErrNotEligible) {
				t.Errorf("MapToDHI(%q) error = %v, want ErrNotEligible", tt.input, err)
			}
			if got != nil {
				t.Errorf("MapToDHI(%q) = %q, want nil", tt.input, got.String())
			}
		})
	}
}
