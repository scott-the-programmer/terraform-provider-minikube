package main

import (
	"flag"
	"fmt"

	"generator/m/v2/builder"
)

func main() {
	targetFile := flag.String("target", "schema_cluster.go", "a string")
	flag.Parse()

	fmt.Println(*targetFile)
	schemaBuilder := builder.NewSchemaBuilder(*targetFile, &builder.MinikubeHostBinary{})
	err := schemaBuilder.Build()
	if err != nil {
		panic(err)
	}
}
