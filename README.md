# Numbers Station

## How to build

Terminal 1:

```
cd server
go mod tidy
go compile main.go
```

Terminal 2:

```
cd client
go mod tidy
go compile main.go
```

## How to run

Terminal 1:

```
cd server
go mod tidy
go run main.go
```

Terminal 2:

```
cd client
go mod tidy
go run main.go
```

## How to test

Test the client reconnection in one of two ways :

1. Run the server with a -failEvery <int> flag and the server with fail with an error when it hits the modulus of that number
2. Run the server with a -failEvery <int> flag and a -goOfflineFor flag and the stream will send an error for that many seconds

Test the checksum generation :

```
cd numbers
go test
```

Test the stale cache (30s expiry)

1. Run a client and server
2. Note the client UUID
3. Terminate the client
4. Wait 30s
5. Run the client with -overrideClientID <clientID>
6. Watch the client fail, and the client not exit when it's supposed to because of the wrapped error

## Limitations

- I couldn't justify spending more time on getting GRPC to fail, I tried with the included iptables scripts as well but the server wouldn't reliably realise the client had gone away
- The numbers are generated upon connection - this is fine for numbers but larger workloads would not be very good (time to initial message would be increased)
- Max concurrent connections is limited by the available memory due to the above
- The code for the sporadically failing server is based on https://github.com/grpc/grpc-go/blob/v1.48.0/examples/features/retry/server/main.go, but because that implementation fails on connection whereas I needed to fail mid-stream, I integrated the maybeFail function into the real server impl. Possibly a DDoS attack vector!

## Improvements

### Numbers

- More test cases? Although this one did expose that I was only CRCing the first byte of every number (before using strconv)

### Repository

- Should the numbers be culled when a tombstone clientID is left? Saves space.
- Should the numbers be generated upon dispatch rather than all-at-once? trade off computation time vs storage space

### Server

- Signal handling for a graceful exit
- A way to refuse connections rather than just return a stream error in order to test the exponential backoff

### Client

- Quite messy, most of the time has been spent trying to work out how to get a GRPC server to misbehave.
- If the client terminates and resends the same clientID then as it's stateless it doesn't have the numbers up to PTS. The server could rewind? Or the client has state too? Anyway, the overrideClientID arg is probably not useful except when checking that a clientID sent in a connection request after 30s is marked as stale and unuseable.