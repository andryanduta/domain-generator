package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
	fmt.Println("Domain abstraction has been created!")

	// Create service
	servicePath := filepath.Join(basePath, "service")
	createDir(servicePath)
	fmt.Println("Domain service has been created!")

	// Initialize store
	if err := StoreInit(basePath, domainName); err != nil {
		return err
	}

	// Initialize handler
	if err := HandlerInit(basePath, domainName); err != nil {
		return err
	}

	return nil
}

// StoreInit initializes the store directory and files.
func StoreInit(basePath, domainName string) error {
	reader := bufio.NewReader(os.Stdin)
	storePath := filepath.Join(basePath, "store")
	createDir(storePath)
	fmt.Println("Store directory has been created!")

	for _, storeType := range []string{"database store", "cache store", "BT store"} {
		if prompt(fmt.Sprintf("Do you wish to install the %s? (Y/N): ", storeType), reader) {
			storeFile := filepath.Join(storePath, fmt.Sprintf("%s.go", strings.ReplaceAll(storeType, " ", "")))
			createFile(storeFile, fmt.Sprintf("// %s template for %s\n", strings.Title(storeType), domainName))
			fmt.Printf("%s has been created!\n", strings.Title(storeType))
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

	for _, handlerType := range []string{"NSQ handler", "CRON handler", "HTTP handler", "GRPC handler"} {
		if prompt(fmt.Sprintf("Do you wish to install the %s? (Y/N): ", handlerType), reader) {
			handlerTypeDir := filepath.Join(handlerPath, strings.ToLower(strings.Split(handlerType, " ")[0]))
			createDir(handlerTypeDir)
			handlerFile := filepath.Join(handlerTypeDir, fmt.Sprintf("%s.go", strings.ReplaceAll(handlerType, " ", "")))
			createFile(handlerFile, fmt.Sprintf("// %s template for %s\n", strings.Title(handlerType), domainName))
			fmt.Printf("%s has been created!\n", strings.Title(handlerType))
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
