package registry

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/cli/environment"
	"github.com/containers/image/v5/types"
)

// Client provides methods for interacting with container registries
type Client struct {
	sysCtx *types.SystemContext
}

// NewClient creates a new registry client
// It respects CONTAINERS_REGISTRIES_CONF environment variable for registry configuration
func NewClient() *Client {
	sysCtx := &types.SystemContext{}

	// Apply CONTAINERS_REGISTRIES_CONF or REGISTRIES_CONFIG_PATH env vars if set
	if err := environment.UpdateRegistriesConf(sysCtx); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load registries config: %v\n", err)
	}

	return &Client{
		sysCtx: sysCtx,
	}
}

// GetDigest resolves an image reference to its digest
func (c *Client) GetDigest(ctx context.Context, ref reference.Named) (string, error) {
	// Add default tag if not present
	if _, ok := ref.(reference.Tagged); !ok {
		if _, ok := ref.(reference.Digested); !ok {
			var err error
			ref, err = reference.WithTag(ref, "latest")
			if err != nil {
				return "", fmt.Errorf("failed to add default tag: %w", err)
			}
		}
	}

	imgRef, err := docker.NewReference(ref)
	if err != nil {
		return "", fmt.Errorf("failed to create docker reference: %w", err)
	}

	imgSrc, err := imgRef.NewImageSource(ctx, c.sysCtx)
	if err != nil {
		return "", fmt.Errorf("failed to create image source for %s: %w", ref.String(), err)
	}
	defer func() { _ = imgSrc.Close() }()

	manifestBytes, _, err := imgSrc.GetManifest(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get manifest for %s: %w", ref.String(), err)
	}

	digest, err := manifest.Digest(manifestBytes)
	if err != nil {
		return "", fmt.Errorf("failed to compute manifest digest for %s: %w", ref.String(), err)
	}

	return digest.String(), nil
}

// CheckAuth verifies that the client can authenticate to a registry.
// It does this by attempting to access an image reference, which triggers auth.
// Returns nil if authentication succeeds (even if image doesn't exist),
// or an error describing the auth failure.
func (c *Client) CheckAuth(ctx context.Context, registry string) error {
	// Try to access a reference to trigger auth
	// The image doesn't need to exist - we just need to verify auth works
	refStr := registry + "/auth-check:test"
	ref, err := reference.ParseNormalizedNamed(refStr)
	if err != nil {
		return fmt.Errorf("invalid registry reference: %w", err)
	}

	imgRef, err := docker.NewReference(ref)
	if err != nil {
		return fmt.Errorf("failed to create docker reference: %w", err)
	}

	// Try to create an image source - this will trigger authentication
	imgSrc, err := imgRef.NewImageSource(ctx, c.sysCtx)
	if err != nil {
		errStr := strings.ToLower(err.Error())

		// 404/not found means auth succeeded but image doesn't exist - that's OK
		if strings.Contains(errStr, "404") ||
			strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "manifest unknown") ||
			strings.Contains(errStr, "name unknown") ||
			strings.Contains(errStr, "unknown name") {
			return nil
		}

		// Check if it's an auth error
		if strings.Contains(errStr, "unauthorized") ||
			strings.Contains(errStr, "authentication required") ||
			strings.Contains(errStr, "denied") ||
			strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "403") {
			return fmt.Errorf("authentication failed for %s: run 'docker login %s' first", registry, registry)
		}
		return fmt.Errorf("failed to connect to %s: %w", registry, err)
	}
	_ = imgSrc.Close()

	return nil
}

// IsNotFoundOrAuthError checks if an error indicates that an image was not found
// or that authentication is required. These are expected errors when checking
// for DHI equivalents that may not exist.
//
// Returns true for:
//   - 404 Not Found (image doesn't exist)
//   - 401 Unauthorized (authentication required)
//   - 403 Forbidden (access denied)
//   - "manifest unknown" errors (common registry response for missing images)
func IsNotFoundOrAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Check for HTTP status codes in error messages
	// The containers/image library typically includes these in error text
	if strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "404") {
		return true
	}

	// Check for common registry error messages (case-insensitive)
	if strings.Contains(errStr, "manifest unknown") ||
		strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "authentication required") ||
		strings.Contains(errStr, "denied") ||
		strings.Contains(errStr, "does not exist") {
		return true
	}

	return false
}
