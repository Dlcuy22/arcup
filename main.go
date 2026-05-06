// Module: main
// Purpose: Entry point for arcup CLI. Delegates all command
// registration and execution to the cmd package.
//
// Dependencies:
//   - cmd: Cobra command tree and flag definitions
package main

import "github.com/dlcuy22/arcup/cmd"

func main() {
	cmd.Execute()
}
