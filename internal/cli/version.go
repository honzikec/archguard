package cli

import "fmt"

func runVersion(_ []string) int {
	fmt.Printf("archguard version=%s commit=%s build_date=%s\n", Version, Commit, BuildDate)
	return 0
}
