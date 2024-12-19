# Domain generator
Go packages for generating domain

This tool will automatically generates a new domain service based on your needs. Previously we were still very repetitive manually creating the domain service, so to minimize the repetition.

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
├── handler [optional]
  ├── http [optional]
    ├── http.go
    ├── foo_handler.go
    ├── error.go
```

## Instructions
### Command example:
To install the packages on your system
```
$ go get github.com/andryanduta/domain-generator
```
Command template
```
$ domain-generator -name [domain_name]
```
Complete example
```
$ domain-generator -name usermanager
```
After the generator runs, you will be asked whether you want install handler, store inside the domain model. Please answer with Y or N (case insensitive).
# Note: 
Domain address and domain name can't be empty or replacing an existing domain is prohibited!