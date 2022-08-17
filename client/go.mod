module client

go 1.19

replace github.com/bmcgavin/numbers => ../numbers

require (
	github.com/bmcgavin/numbers v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.1.2
	google.golang.org/grpc v1.48.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	golang.org/x/sys v0.0.0-20220209214540-3681064d5158 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20201019141844-1ed22bb0c154 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
