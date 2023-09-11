package regoservice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/open-policy-agent/opa/rego"
)

const (
	module = `package hooker

default allow = false

allow {
%s
}
`
	defaultPathToRegoFilters = "./rego-filters"
)

var pathToRegoFilters = ""

func getFilesWithPathToRegoFilters(files []string) []string {
	if pathToRegoFilters == "" {
		if os.Getenv("REGO_FILTERS_PATH") != "" {
			pathToRegoFilters = os.Getenv("REGO_FILTERS_PATH")
		} else {
			pathToRegoFilters = defaultPathToRegoFilters
		}
	}
	filesWithPath := make([]string, len(files))
	copy(filesWithPath, files)
	for i, file := range filesWithPath {
		if !strings.HasPrefix(file, pathToRegoFilters) {
			filesWithPath[i] = filepath.Join(pathToRegoFilters, file)
		}
	}
	return filesWithPath
}

func buildRegoLoader(files []string, rule string) func(r *rego.Rego) {
	if IsUsedRegoFiles(files) {
		filesWithPath := getFilesWithPathToRegoFilters(files)
		return rego.Load(filesWithPath, nil)
	}

	return rego.Module("hooker.rego", fmt.Sprintf(module, rule))
}
func IsUsedRegoFiles(files []string) bool {
	return len(files) != 0 && files[0] != ""
}
func DoesMatchRegoCriteria(input interface{}, files []string, rule string) (bool, error) {
	if !IsUsedRegoFiles(files) && rule == "" {
		return true, nil
	}

	ctx := context.Background()

	r := rego.New(
		rego.Query("x = data.hooker.allow"),
		buildRegoLoader(files, rule),
	)

	query, err := r.PrepareForEval(ctx)
	if err != nil {
		return false, err
	}

	rs, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		return false, err
	}

	if len(rs) > 0 {
		switch rs[0].Bindings["x"].(type) {
		case bool:
			return rs[0].Bindings["x"].(bool), nil
		}
	}
	return false, nil
}
