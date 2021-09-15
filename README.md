# tfget

[![CI](https://github.com/bogdanbarna/tfget/actions/workflows/action.yml/badge.svg)](https://github.com/bogdanbarna/tfget/actions/workflows/action.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/bogdanbarna/tfget)](https://goreportcard.com/report/github.com/bogdanbarna/tfget)
[![codecov](https://codecov.io/gh/bogdanbarna/tfget/branch/main/graph/badge.svg)](https://codecov.io/gh/bogdanbarna/tfget)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://conventionalcommits.org)
[![License](https://img.shields.io/github/license/bogdanbarna/tfget)](/LICENSE)
<!-- [![Release](https://img.shields.io/github/release/bogdanbarna/tfget.svg)](https://github.com/bogdanbarna/tfget/releases/latest) -->

## About

Golang script for fetching a specific Terraform version.
Inspired by rbenv, pyenv, tfenv, etc.

## Usage

### Prerequisites

> go >= 1.16

### Install

```bash
git clone git@github.com:bogdanbarna/tfget.git
cd tfget
go build
./tfget help
```

### Run

```bash
./tfget list-remote
./tfget list-local
./tfget use 1.0.4
./tfget list-local
export PATH="$HOME/.tfget/versions:$PATH"
which terraform
terraform --version
```

## Development

- Docs
- Roadmap
- Contributing

## FAQs
