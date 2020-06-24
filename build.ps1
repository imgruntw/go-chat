cp go-chat.html build/go-chat.html

$env:GOARCH="amd64"

$env:GOOS="windows"
go build -o build/go-chat.exe .

#$env:GOOS="linux"
#go build -o build/go-chat.o .

#$env:GOOS="darwin"
#go build -o build/go-chat.app .
