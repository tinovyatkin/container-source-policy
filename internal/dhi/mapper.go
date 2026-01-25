// Package dhi provides utilities for mapping container image references to
// Docker Hardened Images (dhi.io) equivalents.
package dhi

import (
	"errors"
	"strings"

	"github.com/containers/image/v5/docker/reference"
)

// ErrNotEligible is returned when an image reference cannot be mapped to DHI.
var ErrNotEligible = errors.New("image not eligible for DHI mapping")

const (
	// Registry is the DHI registry hostname
	Registry = "dhi.io"
	// DockerHubDomain is the canonical Docker Hub domain
	DockerHubDomain = "docker.io"
	// LibraryPrefix is the path prefix for official Docker Hub images
	LibraryPrefix = "library/"
)

// CanMapToDHI returns true if the reference is a docker.io library image
// that may have a DHI equivalent.
// Only official images (docker.io/library/*) are eligible for DHI mapping.
func CanMapToDHI(ref reference.Named) bool {
	domain := reference.Domain(ref)
	path := reference.Path(ref)

	// Only docker.io images
	if domain != DockerHubDomain {
		return false
	}

	// Only library images (docker.io/library/*)
	// These are the official images like alpine, node, golang, etc.
	return strings.HasPrefix(path, LibraryPrefix)
}

// MapToDHI converts a docker.io library reference to its dhi.io equivalent.
// Example: docker.io/library/alpine:3.18 -> dhi.io/alpine:3.18
// Returns ErrNotEligible if the reference cannot be mapped (use CanMapToDHI to check first).
func MapToDHI(ref reference.Named) (reference.Named, error) {
	if !CanMapToDHI(ref) {
		return nil, ErrNotEligible
	}

	path := reference.Path(ref)
	// Remove "library/" prefix: library/alpine -> alpine
	dhiPath := strings.TrimPrefix(path, LibraryPrefix)

	// Build dhi.io reference string
	dhiRefStr := Registry + "/" + dhiPath

	// Preserve tag if present
	if tagged, ok := ref.(reference.Tagged); ok {
		dhiRefStr += ":" + tagged.Tag()
	}

	return reference.ParseNormalizedNamed(dhiRefStr)
}
