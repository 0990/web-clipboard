::build
SET GOOS=linux
SET GOARCH=amd64
go build -o bin/web-clipboard cmd/main.go

SET GOOS=windows
SET GOARCH=amd64
go build -o bin/web-clipboard.exe cmd/main.go