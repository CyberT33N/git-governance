# Verification

```powershell
$env:GOTOOLCHAIN = "local"
$env:GOFLAGS = "-mod=readonly"
$env:GOVCS = "*:off"
go test -mod=readonly ./...
go run -mod=readonly .\cmd\check-coverage
CGO_ENABLED=1 go test -mod=readonly -race ./...
go vet -mod=readonly ./...
go test -mod=readonly ./internal/integration -count=1
go tool -modfile tools/go.mod govulncheck ./...
```

`check-coverage` executes `go test -count=1 -cover ./...`. It fails when a Go
package has no `_test.go` file or its executable statements do not reach
`100.0%` coverage.

On Windows, the race detector needs a working C compiler and `CGO_ENABLED=1`.
If the local host cannot provide that compiler, use the Ubuntu CI gate rather
than claiming a skipped race run passed.

Fuzz smoke tests:

```powershell
go test ./internal/domain/ticket -run=^$ -fuzz=FuzzParseTicketValues -fuzztime=2s -parallel=1
go test ./internal/domain/branch -run=^$ -fuzz=FuzzParseBranchValues -fuzztime=2s -parallel=1
go test ./internal/domain/commitmsg -run=^$ -fuzz=FuzzParseCommitMessage -fuzztime=2s -parallel=1
go test ./internal/adapters/configfs -run=^$ -fuzz=FuzzDecodePreferences -fuzztime=2s -parallel=1
go test ./internal/adapters/quality -run=^$ -fuzz=FuzzDecodeQualityConfiguration -fuzztime=2s -parallel=1
```
