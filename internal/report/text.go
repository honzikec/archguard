package report

import (
	"fmt"
	"github.com/honzikec/archguard/internal/model"
)

func PrintText(findings []model.Finding) {
	if len(findings) == 0 {
		return
	}
	for _, f := range findings {
		fmt.Printf("%s\n\n%s\nimports\n%s\n\n", f.Message, f.FilePath, f.RawImport)
	}
}
