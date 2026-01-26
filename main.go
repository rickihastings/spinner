package main

import (
	"fmt"
	"github.com/rickihastings/spinner/internal/prerequisites"
)

func main() {
	if err := prerequisites.CheckPrerequisites(); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}
	fmt.Println("All prerequisites checked successfully")
}
