build:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/grpc/messenger.proto
	go build cmd/app.go
	go build cmd/client.go

clean:
	rm app.exe app