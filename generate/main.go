package main

import (
	"flag"
	"fmt"

	"terraform-provider-minikube/m/v2/minikube/generator"
)

func main() {
	targetFile := flag.String("target", "schema_cluster.go", "a string")
	flag.Parse()

	fmt.Println(*targetFile)
	schemaBuilder := generator.NewSchemaBuilder(*targetFile, &generator.MinikubeHostBinary{})
	schema, err := schemaBuilder.Build()
	if err != nil {
		panic(err)
	}

	err = schemaBuilder.Write(schema)
	if err != nil {
		panic(err)
	}

}
