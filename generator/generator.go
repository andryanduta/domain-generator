package generator

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Embed the template files
//go:embed template/domain.go.tmpl
var domainTemplate string

//go:embed template/service.go.tmpl
var serviceTemplate string

//go:embed template/store/store.go.tmpl
var storeTemplate string

//go:embed template/store/cachestore.go.tmpl
var storeCacheTemplate string

//go:embed template/handler/http/http.go.tmpl
var httpHandlerTemplate string

//go:embed template/handler/http/error.go.tmpl
var httpErrorTemplate string

//go:embed template/handler/http/foo_handler.go.tmpl
var httpFooHandlerTemplate string

//go:embed template/handler/http/util.go.tmpl
var httpUtilTemplate string

// GenerateDomain creates the domain abstraction, service, store, and handler directories and files in the current working directory.
func GenerateDomain(domainName string) error {
	if domainName == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	// Get the user's current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}

	basePath := filepath.Join(currentDir, domainName)

	if _, err := os.Stat(basePath); !os.IsNotExist(err) {
		return fmt.Errorf("domain already exists in the current directory")
	}

	// Create domain abstraction
	createDir(basePath)
	outputFile := filepath.Join(basePath, domainName+".go")
	generateFileFromTemplate(domainTemplate, outputFile, domainName)
	fmt.Println("Domain abstraction has been created!")

	// Create service
	servicePath := filepath.Join(basePath, "service")
	createDir(servicePath)
	outputFile = filepath.Join(servicePath, "service.go")
	generateFileFromTemplate(serviceTemplate, outputFile, domainName)

	fmt.Println("Domain service has been created!")

	// Initialize store
	if err := StoreInit(basePath, domainName, servicePath); err != nil {
		return err
	}

	// Initialize handler
	if err := HandlerInit(basePath, domainName); err != nil {
		return err
	}

	return nil
}

// StoreInit initializes the store directory and files, including DB store, cache store, and BT store.
func StoreInit(basePath, domainName, servicePath string) error {
	reader := bufio.NewReader(os.Stdin)
	storePath := filepath.Join(basePath, "store")
	createDir(storePath)
	fmt.Println("Store directory has been created!")

	for _, storeType := range []string{"database store", "cache store"} {
		if prompt(fmt.Sprintf("Do you wish to install the %s? (Y/N): ", storeType), reader) {

			switch storeType {
			case "database store":
				// Generate DB store files
				outputFile := filepath.Join(storePath, "store.go")
				generateFileFromTemplate(storeTemplate, outputFile, domainName)

				// Append DB store to service file
				if err := appendStoreToService("store", servicePath, domainName); err != nil {
					return fmt.Errorf("failed to append DB store to service: %v", err)
				}

			case "cache store":
				// Generate Cache store files
				outputFile := filepath.Join(storePath, "cachestore.go")
				generateFileFromTemplate(storeCacheTemplate, outputFile, domainName)

				// Append Cache store to service file
				if err := appendStoreToService("cacheStore", servicePath, domainName); err != nil {
					return fmt.Errorf("failed to append Cache store to service: %v", err)
				}
			}

			fmt.Printf("%s has been created!\n", storeType)
		}
	}
	return nil
}

// HandlerInit initializes the handler directory and files.
func HandlerInit(basePath, domainName string) error {
	reader := bufio.NewReader(os.Stdin)
	handlerPath := filepath.Join(basePath, "handler")
	createDir(handlerPath)
	fmt.Println("Handler directory has been created!")

	for _, handlerType := range []string{"HTTP handler"} {
		if prompt(fmt.Sprintf("Do you wish to install the %s? (Y/N): ", handlerType), reader) {

			switch handlerType {
			case "HTTP handler":
				httpPath := filepath.Join(handlerPath, "http")
				createDir(httpPath)

				// Generate http files
				outputFile := filepath.Join(httpPath, "http.go")
				generateFileFromTemplate(httpHandlerTemplate, outputFile, domainName)

				outputFile = filepath.Join(httpPath, "error.go")
				generateFileFromTemplate(httpErrorTemplate, outputFile, domainName)

				outputFile = filepath.Join(httpPath, "foo_handler.go")
				generateFileFromTemplate(httpFooHandlerTemplate, outputFile, domainName)

				outputFile = filepath.Join(httpPath, "util.go")
				generateFileFromTemplate(httpUtilTemplate, outputFile, domainName)
			}

			fmt.Printf("%s has been created!\n", handlerType)
		}
	}
	return nil
}

func prompt(question string, reader *bufio.Reader) bool {
	fmt.Print(question)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func createDir(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		fmt.Printf("Error creating directory %s: %v\n", path, err)
		os.Exit(1)
	}
}

func createFile(filePath, content string) {
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		fmt.Printf("Error creating file %s: %v\n", filePath, err)
		os.Exit(1)
	}
}

func generateFileFromTemplate(templateContent, outputPath, domainName string) error {
	content := strings.ReplaceAll(templateContent, "<%= domainname %>", domainName)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}
	return nil
}

func appendStoreToService(storeType, servicePath, domainName string) error {
	serviceFile := filepath.Join(servicePath, "service.go")

	// Read the service file content
	content, err := os.ReadFile(serviceFile)
	if err != nil {
		return fmt.Errorf("failed to read service file: %v", err)
	}
	lines := strings.Split(string(content), "\n")

	var updatedLines []string
	mockgenAdded := false
	newFuncUpdated := false
	serviceStructUpdated := false

	for _, line := range lines {

		// Add Stores interfaces with mockgen directives
		if !mockgenAdded && strings.Contains(line, "//<FOR_STORE_CODE_GENERATION>") {
			mockgenAdded = true
			updatedLines = append(updatedLines, line)

			if storeType == "store" {

				updatedLines = append(updatedLines,
					fmt.Sprintf("\n//go:generate mockgen -destination ./mock_store_test.go -package service github.com/yourreponame/yourpath/v2/%s/service Store", domainName),
					"",
					fmt.Sprintf("// Store denotes the %s store.", domainName),
					"type Store interface {",
					"\tNewClient(usingTx bool) (StoreClient, error)",
					"\tNewClientWithTx(tx *sqlx.Tx) StoreClient",
					"}",

					"",
					"// StoreClient provides mechanism to access persistent store.",
					"type StoreClient interface {",
					"\tCommit() error",
					"\tRollback() error",
					"\t// add your store interface here",
					"}",
				)
			} else if storeType == "cacheStore" {

				updatedLines = append(updatedLines,
					fmt.Sprintf("\n//go:generate mockgen -destination ./mock_cachestore_test.go -package service github.com/yourreponame/yourpath/v2/%s/service CacheStore", domainName),
					"",
					fmt.Sprintf("// CacheStore provides mechanism to access %s cache store", domainName),
					"type CacheStore interface{",
					"\t//add your cache store interface here",
					"}",
				)
			}
			continue
		}

		// Add the store, cacheStore, or bigTable fields to the Service struct
		if !serviceStructUpdated && strings.Contains(line, "type Service struct {") {
			serviceStructUpdated = true
			updatedLines = append(updatedLines, line)
			if storeType == "store" {
				updatedLines = append(updatedLines, "\tstore Store")
			} else if storeType == "cacheStore" {
				updatedLines = append(updatedLines, "\tcacheStore CacheStore")
			}

			continue
		}

		// Update the New function to include the new store parameters
		if !newFuncUpdated && strings.Contains(line, "func New") {
			newFuncUpdated = true
			// Find the opening and closing parentheses of the parameter list
			openParenIdx := strings.Index(line, "(")
			closeParenIdx := strings.Index(line, ")")
			beforeParen := line[:openParenIdx+1] // Everything before and including "("
			params := strings.TrimSpace(line[openParenIdx+1 : closeParenIdx])
			afterParen := line[closeParenIdx:] // Everything after ")"

			// Append the new parameter correctly
			if params == "" {
				// No existing parameters
				if storeType == "store" {
					updatedLines = append(updatedLines, beforeParen+"store Store"+afterParen)
				} else if storeType == "cacheStore" {
					updatedLines = append(updatedLines, beforeParen+"cacheStore CacheStore"+afterParen)
				}
			} else {
				// Existing parameters
				if storeType == "store" {
					updatedLines = append(updatedLines, beforeParen+params+", store Store"+afterParen)
				} else if storeType == "cacheStore" {
					updatedLines = append(updatedLines, beforeParen+params+", cacheStore CacheStore"+afterParen)
				}
			}
			continue
		}

		// Add the store assignment to the Service struct in the New function
		if newFuncUpdated && strings.Contains(line, "return &Service{") {
			updatedLines = append(updatedLines, line)
			if storeType == "store" {
				updatedLines = append(updatedLines, "\t\tstore: store,")
			} else if storeType == "cacheStore" {
				updatedLines = append(updatedLines, "\t\tcacheStore: cacheStore,")
			}

			continue
		}

		// Append remaining lines unchanged
		updatedLines = append(updatedLines, line)
	}

	// Write updated content back to the service file
	return os.WriteFile(serviceFile, []byte(strings.Join(updatedLines, "\n")), 0644)
}
