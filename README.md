# Golang Domain Generator
A Go tool for automatically generating domain-driven design (DDD) based domain services. 

The Domain Generator tool helps you quickly create a domain service structure, minimizing the need for repetitive boilerplate code. By using this tool, you can define the domain, service, store, and handler components more efficiently.

## Features
- Automatically generates domain services.
- Supports the creation of optional store and handler components.
- Easily configurable with interactive prompts for adding handlers and stores

## Architecture
```
domain-name
├── domain-name.go: The main domain abstraction file.
├── service
  ├── service.go: The service file for the domain.
├── store
  ├── store.go: The database store implementation.
  ├── cachestore.go: The cache store implementation.
├── handler
  ├── http
    ├── http.go: Main HTTP handler.
    ├── foo_handler.go: Example HTTP handler.
    ├── error.go: Error handling for HTTP requests.
```

## Installation

To install the tool, run:

```
$ go install github.com/andryanduta/domain-generator@latest
```

## Usage
Basic command
```
$ domain-generator -name [domain_name]
```

Example
To generate a domain for usermanager, run the following command:
```
$ domain-generator -name usermanager
```
This will create the directory usermanager/ with the appropriate Go files. After the generator runs, you will be prompted to decide if you want to install optional components (handlers, stores, etc.) within the domain model. You can respond with Y or N (case-insensitive) for each prompt.

Notes
- Domain Name: The domain name cannot be empty. It must be a valid name for your domain.
- No Overwriting: You cannot replace an existing domain directory with the same name. Ensure the domain name is unique in the current directory.

### How it works
- Domain Creation: The tool creates a domain abstraction file (domain-name.go) for the specified domain.
- Service Generation: It generates a service file that will be used to handle business logic for the domain.
- Store Integration: Optionally, it can generate a store (either database or cache) within the store/ directory.
- Handler Integration: Optionally, it can create an HTTP handler structure in the handler/ directory for your domain.

### Example Flow
```
$ domain-generator -name usermanager

Domain generation in progress...
Would you like to install the HTTP handler? (Y/N): Y
Would you like to install the store (database)? (Y/N): Y
Would you like to install the cache store? (Y/N): N

Domain 'usermanager' has been created!
```

## Contributing
Feel free to contribute to the project by submitting issues, feature requests, or pull requests. If you have any improvements or suggestions, please open an issue or create a pull request.

## License
This project is licensed under the MIT License - see the LICENSE file for details.
