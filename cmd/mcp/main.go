// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main is the main package for the standalone CEL MCP server.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cel-expr/skills/internal/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	fileDescriptorSet := flag.String("file_descriptor_set", "", "Path to a binary FileDescriptorSet file containing protobuf definitions.")
	flag.String("file_descriptors", "", "Alias for -file_descriptor_set: path to a binary FileDescriptorSet file.")
	environmentFlag := flag.String("environment", "", "Path to a JSON file or JSON string containing the CEL environment configuration.")
	flag.String("env", "", "Alias for -environment: path to a JSON file or JSON string containing the CEL environment configuration.")
	flag.Parse()

	fdsPath := *fileDescriptorSet
	if fdsPath == "" {
		if f := flag.Lookup("file_descriptors"); f != nil {
			fdsPath = f.Value.String()
		}
	}

	envPath := *environmentFlag
	if envPath == "" {
		if f := flag.Lookup("env"); f != nil {
			envPath = f.Value.String()
		}
	}

	s, err := mcp.SetupServer(fdsPath, envPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup error: %v\n", err)
		os.Exit(1)
	}

	if err := s.Run(context.Background(), &sdk.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
