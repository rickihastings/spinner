package prerequisites

import (
	"os/exec"
)

type Prerequisite struct {
	Name         string
	Command      string
	ErrorMessage string
}

var prerequisites = []Prerequisite{
	{
		Name:         "Docker",
		Command:      "docker",
		ErrorMessage: "Docker is not installed. Please install Docker Desktop.",
	},
	{
		Name:         "Git",
		Command:      "git",
		ErrorMessage: "Git is not installed. Please install Git.",
	},
	{
		Name:         "Claude",
		Command:      "claude",
		ErrorMessage: "Claude CLI is not installed. Please install claude-code.",
	},
}

type PrerequisiteError struct {
	Prerequisite Prerequisite
}

func (e *PrerequisiteError) Error() string {
	return e.Prerequisite.ErrorMessage
}

func CheckPrerequisites() error {
	for _, prereq := range prerequisites {
		cmd := exec.Command(prereq.Command, "--version")
		if err := cmd.Run(); err != nil {
			return &PrerequisiteError{Prerequisite: prereq}
		}
	}
	return nil
}
