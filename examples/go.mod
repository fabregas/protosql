module examples

go 1.17

replace github.com/fabregas/protosql => ../

require (
	github.com/fabregas/protosql v0.0.2
	github.com/lib/pq v1.10.4
	github.com/sirupsen/logrus v1.8.1
	google.golang.org/protobuf v1.27.1
)

require golang.org/x/sys v0.0.0-20191026070338-33540a1f6037 // indirect
