set dotenv-load := false
set shell := ["sh", "-cu"]

BIN := justfile_directory() / ".bin"

[private]
default:
    @just --list

# ---- golangci-lint

GOLANGCI_LINT_VERSION := 'v2.5.0'
GOLANGCI_LINT_PATH := BIN / 'golangci-lint'
GOLANGCI_LINT := GOLANGCI_LINT_PATH + '@' + GOLANGCI_LINT_VERSION

[private]
install-golangci-lint:
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b {{ BIN }} {{ GOLANGCI_LINT_VERSION }}
    mv {{ GOLANGCI_LINT_PATH }} {{ GOLANGCI_LINT }}

[doc('Run static analysis using all available linters to detect code issues')]
[group('code')]
lint:
    @if test ! -e {{ GOLANGCI_LINT }}; then just install-golangci-lint; fi
    {{ GOLANGCI_LINT }} run ./...

# ---- fieldaligment

FIELDALIGNMENT_VERSION := 'v0.38.0'
FIELDALIGNMENT_PATH := BIN / 'fieldalignment'
FIELDALIGNMENT := FIELDALIGNMENT_PATH + '@' + FIELDALIGNMENT_VERSION

[private]
install-fieldaligment:
    GOBIN={{ BIN }} go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@{{ FIELDALIGNMENT_VERSION }}
    mv {{ FIELDALIGNMENT_PATH }} {{ FIELDALIGNMENT }}

[doc('Reorder struct fields to improve memory layout and reduce padding')]
[group('code')]
align:
    @if test ! -e {{ FIELDALIGNMENT }}; then just install-fieldaligment; fi
    {{ FIELDALIGNMENT }} --fix ./...

# ---- testing

[private]
smoke:
    go test ./...

[private]
cover:
    go test -count 1 -parallel 4 -race -coverprofile=coverage.out
    go tool cover -func coverage.out

# ---- shortcuts

[doc('Run all code quality tools: struct alignment and static analysis')]
[group('code')]
code: align lint

[doc('Run tests by type: `smoke` for quick checks, `cover` for detailed analysis')]
[group('test')]
test type='smoke':
    just {{ type }}
