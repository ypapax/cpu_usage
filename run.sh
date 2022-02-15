set -ex
go test -v
exe=cpu_usage
go build -o $GOPATH/bin/$exe
$exe