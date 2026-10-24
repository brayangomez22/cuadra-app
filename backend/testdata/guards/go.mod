// A module made to break the depguard and forbidigo rules of .golangci.yml
// on purpose:
// make lint-guards runs golangci-lint here and fails unless every rule
// reports its forbidden case.
module github.com/brayangomez22/cuadra-app/backend

go 1.27.3

require github.com/google/uuid v1.6.0
