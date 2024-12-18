package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/andryanduta/domain-generator/generator"
)

func main() {
	help := flag.Bool("h", false, "Display help")
	domainName := flag.String("name", "", "Domain name to generate (required)")

	flag.Parse()

	if *help {
		printHelp()
		return
	}

	if *domainName == "" {
		fmt.Println("Error: Domain name is required.")
		printHelp()
		os.Exit(1)
	}

	fmt.Println("Starting Domain Generator...")

	err := generator.GenerateDomain(*domainName)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}

	fmt.Println("Domain generation complete! Files have been created in your current directory.")
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  domain-generator -name [domain_name]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h          Show help")
	fmt.Println("  -name       Name of the domain to generate (required)")
}
