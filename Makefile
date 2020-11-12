build:
	go build cmd/app.go

grpc:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/grpc/messenger.proto

cert:
	openssl genrsa -out ./openssl/ca.key 4096
	openssl req -new -x509 -key ./openssl/ca.key -sha256 -subj "/C=JP/ST=TOK/L=Academy City/O=SYSTEM/OU=Level 6 Shift" -days 365 -out ./openssl/ca.cert
	openssl genrsa -out ./openssl/service.key 4096
	openssl req -new -key ./openssl/service.key -out ./openssl/service.csr -config ./openssl/certificate.conf
	openssl x509 -req -in ./openssl/service.csr -CA ./openssl/ca.cert -CAkey ./openssl/ca.key -CAcreateserial -out ./openssl/service.pem -days 365 -sha256 -extfile ./openssl/certificate.conf -extensions req_ext

clean:
	rm app.exe app