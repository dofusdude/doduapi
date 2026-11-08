package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/dofusdude/doduapi/query"
)

func main() {
	output := flag.String("output", "", "Write the query contract to this file")
	flag.Parse()
	body, err := json.MarshalIndent(query.Endpoints, "", "  ")
	if err != nil {
		panic(err)
	}
	body = append(body, '\n')
	if *output == "" {
		fmt.Print(string(body))
		return
	}
	if err := os.WriteFile(*output, body, 0644); err != nil {
		panic(err)
	}
}
