# Domain generator
Go packages for generating domain

Previously we were still very repetitive manually creating the domain service, so to minimize the repetition, this tool will automatically generates a new domain service based on your needs. 

# Architecture
```
domain-name
├── domain-name.go
├── service
  ├── service.go
  ├── service_test.go
├── store [optional]
  ├── store.go 
  ├── store_test.go
  ├── cachestore.go
  ├── bt_store.go
├── handler [optional]
  ├── http [optional]
    ├── http.go
    ├── foo_handler.go
    ├── error.go
  ├── nsq [optional]
    ├── nsq.go
    ├── handler.go
  ├── grpc [optional]
    ├── grpc.go
    ├── handler.go
    ├── error.go
  ├── cron [optional]
    ├── cron.go
```

## Instructions
### Command example:
To install the packages on your system
```
$ go get github.com/andryanduta/domain-generator
```
Command template
```
$ ./generator.sh [domain_name]
```
Complete example
```
$ ./generator.sh adexperiment
```
After the generator runs, you will be asked whether you want install handler, store inside the domain model. Please answer with Y or N (case insensitive).
# Note: 
Domain address and domain name can't be empty or replacing an existing domain is prohibited!