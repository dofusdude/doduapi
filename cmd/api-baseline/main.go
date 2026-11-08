package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/dofusdude/doduapi/internal/apitest"
)

func main() {
	directory := flag.String("data", "", "Directory containing MAPPED_*.json")
	output := flag.String("output", "", "Baseline file to create after reviewing the source data")
	flag.Parse()
	if *directory == "" || *output == "" {
		flag.Usage()
		os.Exit(2)
	}
	data, err := apitest.Load(*directory)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, err := json.MarshalIndent(apitest.Snapshot(data), "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*output, append(body, '\n'), 0644); err != nil {
		panic(err)
	}
	for _, collection := range data.Collections {
		fmt.Printf("%s: %d\n", collection.Path, len(collection.Records))
	}
}
