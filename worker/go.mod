module worker-project/worker

go 1.21

require (
	github.com/aws/aws-lambda-go v1.51.1
	gopkg.in/yaml.v3 v3.0.1
	worker-project/shared v0.0.0
)

replace worker-project/shared => ../shared

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/redis/go-redis/v9 v9.17.2 // indirect
)
