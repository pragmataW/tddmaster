# tddmaster

`tddmaster` is a Go CLI for state-driven, TDD-oriented agent orchestration.

It is the Go and TDD-specialized counterpart of [`github.com/eser/stack`](https://github.com/eser/stack), where the main project flow is centered around `noskills`. This repository takes that core idea and adapts it for a Go implementation with a TDD-first workflow.

## Install

```bash
go install github.com/pragmataW/tddmaster@latest
```

## Development

```bash
go test ./...
```

## Releases

This repository ships with GoReleaser configuration so tagged releases can publish installable binaries for `tddmaster`.
