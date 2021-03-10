// Copyright 2021 Google LLC
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

// Package version prints the version of this tool, as provided at compile time.
package version

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/google/k8s-digester/pkg/version"
)

var (
	writer = os.Stdout

	// Cmd is the version sub-command
	Cmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			return printVersion()
		},
	}
)

func printVersion() error {
	_, err := fmt.Fprintln(writer, version.Version)
	return err
}
