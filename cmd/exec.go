package cmd

// execCmd is the production exec command
var execCmd = NewExecCommand()

func init() {
	rootCmd.AddCommand(execCmd)
}
