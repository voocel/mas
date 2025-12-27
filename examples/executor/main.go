package main

import (
	"context"
	"fmt"
	"os"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/executor/local"
	"github.com/voocel/mas/schema"
)

func main() {
	path := os.Getenv("MAS_SANDBOXD")
	if path == "" {
		path = "mas-sandboxd"
	}

	exec := local.NewLocalExecutor(path)
	call := schema.ToolCall{
		ID:   "1",
		Name: "calculator",
		Args: []byte(`{"expression":"1+1"}`),
	}

	result, err := exec.Execute(context.Background(), call, executor.Policy{
		AllowedTools: []string{"calculator"},
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(string(result.Result))
}
