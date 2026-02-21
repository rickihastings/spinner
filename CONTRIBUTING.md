# Contributing to Spinner

Spinner is primarily a personal project, but contributions are welcome in the right form.

## What's Welcome

- **Bug fixes** — open a PR directly
- **New backends** (e.g. Kubernetes, EC2) — follow the proposal process below
- **New agent types** — follow the proposal process below
- **Small, self-contained improvements** — open a PR with a clear description

## Proposal Process for Larger Changes

For anything that touches public APIs, introduces a new backend, changes the iteration loop or provider interface, or
is otherwise non-trivial, please use the OpenSpec proposal process before writing code:

1. Create a spec with the OpenSpec commands, proposing the new change
2. Open a PR with just the spec — no implementation yet
3. Wait for the spec to be reviewed and merged
4. Implement the change in a follow-up PR referencing the approved spec

This prevents large PRs being rejected on design grounds after significant effort has been spent.

## AI Contributions

AI-generated contributions are welcome. Before submitting, please ensure the code has been reviewed for:

- **Unused functions or variables** — remove them, don't leave dead code
- **Duplicated logic** — consolidate into shared helpers where it makes sense
- **Incorrect patterns** — follow the conventions in `docs/standards.md` and `docs/system-design.md`
- **Missing tests** — all functionality must have tests (see `docs/testing.md`)
- **Major architectural shifts** — keep in line with existing architecture patterns

## Development Setup

See [docs/development.md](docs/development.md) for the full workflow.

## Running Integration Tests

Unit tests run fast and require no external services. Integration tests require a running Docker daemon (Docker tests)
or GCP credentials (GCP tests).

### GCP integration tests

GCP tests are slow. On the first run they bake a VM image, which takes **20–30 minutes**. Subsequent runs reuse the
image and take around **5 minutes**.

Before running, copy `.env.example` to `.env` and fill in the GCP values:

```bash
cp .env.example .env
```

The required variables:

```env
SPINNER_TEST_GCP_PROJECT=your-gcp-project-id
SPINNER_TEST_GCP_ZONE=us-central1-a
SPINNER_TEST_GCP_BUCKET=your-state-bucket
SPINNER_TEST_SERVICE_ACCOUNT=your-service-account@your-project.iam.gserviceaccount.com
```

You also need GCP credentials configured:

```bash
gcloud auth application-default login
# or
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
```

Then run:

```bash
make test-gcp
```

The baked image is kept between runs by default. To force a rebuild:

```bash
SPINNER_TEST_FORCE_REBUILD=1 make test-gcp
```

To delete the image after the run:

```bash
SPINNER_TEST_DELETE_IMAGE=1 make test-gcp
```

## Code Standards

See [docs/standards.md](docs/standards.md) for coding conventions, Go style, and commit message format.

## Questions

Open an issue or start a discussion — happy to give feedback on an idea before you invest time in a spec or
implementation.
