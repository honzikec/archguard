package report

import (
	"encoding/json"
	"fmt"
	"github.com/honzikec/archguard/internal/model"
)

func PrintJSON(findings []model.Finding) {
	if findings == nil {
		findings = []model.Finding{}
	}
	data, _ := json.MarshalIndent(findings, "", "  ")
	fmt.Println(string(data))
}
