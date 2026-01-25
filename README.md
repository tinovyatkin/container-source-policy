# container-source-policy

[![codecov](https://codecov.io/gh/tinovyatkin/container-source-policy/graph/badge.svg?token=tSSxWyOmP2)](https://codecov.io/gh/tinovyatkin/container-source-policy)

Generate BuildKit **source policies** that make Docker builds reproducible and secure — without modifying your Dockerfiles.

- 📌 **Pin** images and URLs to immutable checksums
- 🛡️ **Harden** builds with [Docker Hardened Images](https://dhi.io) (fewer CVEs, smaller footprint)
- ✅ **Validate** existing policies against Dockerfiles *(coming soon)*

See the [BuildKit documentation on build reproducibility](https://github.com/moby/buildkit/blob/master/docs/build-repro.md) for more details on source policies.

## Quick start

```bash
container-source-policy pin --stdout Dockerfile > source-policy.json
EXPERIMENTAL_BUILDKIT_SOURCE_POLICY=source-policy.json docker buildx build -t my-image:dev .
```

> **Note:** [`EXPERIMENTAL_BUILDKIT_SOURCE_POLICY`](https://docs.docker.com/build/building/variables/#experimental_buildkit_source_policy) is the environment variable used by Docker Buildx to specify a source policy file.

## Install

Run directly without installing (recommended):

```bash
# npm/bun
npx container-source-policy --help
bunx container-source-policy --help

# Python
uvx container-source-policy --help

# Ruby (requires RubyGems 3.3+)
gem exec container-source-policy --help
```

Or install globally:

```bash
# Go (build from source)
go install github.com/tinovyatkin/container-source-policy@latest

# npm
npm i -g container-source-policy

# Python
pipx install container-source-policy

# Ruby
gem install container-source-policy
```

## Usage

Generate a policy for one or more Dockerfiles:

```bash
container-source-policy pin --stdout Dockerfile Dockerfile.ci > source-policy.json
```

Read the Dockerfile from stdin:

```bash
cat Dockerfile | container-source-policy pin --stdout -
```

Write directly to a file:

```bash
container-source-policy pin --output source-policy.json Dockerfile
```

### Docker Hardened Images (DHI)

Use `--prefer-dhi` to pin Docker Hub library images to their [Docker Hardened Images](https://www.docker.com/blog/docker-hardened-images-now-free/) equivalents when available:

```bash
# First, login to dhi.io with your Docker Hub credentials
docker login dhi.io

# Then use --prefer-dhi to prefer hardened images
container-source-policy pin --prefer-dhi --stdout Dockerfile
```

This converts eligible images (e.g., `alpine:3.21`, `node:22`, `golang:1.23`) to their `dhi.io` equivalents, which are minimal, security-hardened versions with fewer vulnerabilities.

- Only Docker Hub library images (`alpine`, `node`, `golang`, etc.) are eligible
- Images not available on dhi.io silently fall back to docker.io
- Non-library images (`ghcr.io/*`, `docker.io/myorg/*`) are unchanged
- The policy selector still matches the original reference, so your Dockerfile works unchanged

Example output with `--prefer-dhi`:
```json
{
  "selector": { "identifier": "docker-image://golang:1.23" },
  "updates": { "identifier": "docker-image://dhi.io/golang:1.23@sha256:..." }
}
```

Then pass the policy to BuildKit / Buildx via the environment variable:

```bash
EXPERIMENTAL_BUILDKIT_SOURCE_POLICY=source-policy.json docker buildx build .
```

Or use `buildctl` directly with the `--source-policy-file` flag:

```bash
buildctl build --frontend dockerfile.v0 --local dockerfile=. --local context=. --source-policy-file source-policy.json
```

## What gets pinned

### Container images (`FROM`, `COPY --from`, `ONBUILD`)

- Looks at `FROM …`, `COPY --from=<image>`, and their `ONBUILD` variants across all provided Dockerfiles.
- Skips:
  - `FROM scratch`
  - `FROM <stage>` / `COPY --from=<stage>` references to a previous named build stage
  - `COPY --from=0` numeric stage indices
  - `FROM ${VAR}` / `COPY --from=${VAR}` (unexpanded ARG/ENV variables)
  - images already written as `name@sha256:…`
- Resolves the image manifest digest from the registry and emits BuildKit `CONVERT` rules of the form:
  - `docker-image://<as-written-in-Dockerfile>` → `docker-image://<normalized>@sha256:…`

### HTTP sources (`ADD`, `ONBUILD ADD`)

- Looks at `ADD <url> …` and `ONBUILD ADD <url> …` instructions with HTTP/HTTPS URLs.
- Skips:
  - `ADD --checksum=… <url>` (already pinned)
  - URLs containing unexpanded variables (`${VAR}`, `$VAR`)
  - Git URLs (handled separately, see below)
  - Volatile content (emits warning): URLs returning `Cache-Control: no-store`, `no-cache`, `max-age=0`, or expired `Expires` headers
- Fetches the checksum and emits `CONVERT` rules with `http.checksum` attribute.
- **Respects `Vary` header**: captures request headers that affect response content (e.g., `User-Agent`, `Accept-Encoding`) and includes them in the
  policy as `http.header.*` attributes to ensure reproducible builds.

**Optimized checksum fetching** — avoids downloading large files when possible:

- `raw.githubusercontent.com`: extracts SHA256 from ETag header
- GitHub releases: uses the API `digest` field (set `GITHUB_TOKEN` for higher rate limits)
- S3: uses `x-amz-checksum-sha256` response header (by sending `x-amz-checksum-mode: ENABLED`)
- Fallback: downloads and computes SHA256

### Git sources (`ADD`, `ONBUILD ADD`)

- Looks at `ADD <git-url> …` and `ONBUILD ADD <git-url> …` instructions with Git repository URLs.
- Supports various Git URL formats:
  - `https://github.com/owner/repo.git#ref`
  - `git://host/path#ref`
  - `git@github.com:owner/repo#ref`
  - `ssh://git@host/path#ref`
- Skips URLs containing unexpanded variables (`${VAR}`, `$VAR`)
- Uses `git ls-remote` to resolve the ref (branch, tag, or commit) to a commit SHA
- Emits `CONVERT` rules with `git.checksum` attribute (full 40-character commit SHA)

Example: `ADD https://github.com/cli/cli.git#v2.40.0 /dest` pins to commit `54d56cab...`

## Development

```bash
make build
make test
make lint
```

Update integration-test snapshots:

```bash
UPDATE_SNAPS=true go test ./internal/integration/...
```

## Repository layout

- `cmd/container-source-policy/cmd/`: CLI commands (urfave/cli)
- `internal/dockerfile`: Dockerfile parsing (`FROM` and `ADD` extraction)
- `internal/registry`: registry client (image digest resolution)
- `internal/dhi`: Docker Hardened Images reference mapping
- `internal/http`: HTTP client (URL checksum fetching with optimizations)
- `internal/git`: Git client (commit SHA resolution via git ls-remote)
- `internal/policy`: BuildKit source policy types and JSON output
- `internal/pin`: orchestration logic for `pin`
- `internal/integration`: end-to-end tests with mock registry/HTTP server and snapshots
- `packaging/`: wrappers for publishing prebuilt binaries to npm / PyPI / RubyGems

## Packaging

See `packaging/README.md` for how the npm/PyPI/Ruby packages are assembled from GoReleaser artifacts.
