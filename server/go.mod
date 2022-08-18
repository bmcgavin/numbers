module server

go 1.19

replace github.com/bmcgavin/numbers => ../numbers

replace github.com/bmcgavin/repository => ../repository

require (
	github.com/bmcgavin/numbers v0.0.0-00010101000000-000000000000
	github.com/bmcgavin/repository v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.1.2
	google.golang.org/grpc v1.48.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20210119212857-b64e53b001e4 // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
