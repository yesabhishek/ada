package evals

func DefaultScenarios() []Scenario {
	return []Scenario{
		{
			Name:          "go_disjoint_functions",
			Description:   "Both branches edit different top-level functions in the same file.",
			Language:      "go",
			Path:          "main.go",
			Tags:          []string{"merge", "go", "tie"},
			ExpectVerdict: "tie",
			Base: `package main

func alpha() int {
	return 1
}

func beta() int {
	return 2
}
`,
			Ours: `package main

func alpha() int {
	return 10
}

func beta() int {
	return 2
}
`,
			Theirs: `package main

func alpha() int {
	return 1
}

func beta() int {
	return 20
}
`,
		},
		{
			Name:          "go_same_symbol_conflict",
			Description:   "Both branches change the same function return value differently.",
			Language:      "go",
			Path:          "main.go",
			Tags:          []string{"merge", "go", "conflict"},
			ExpectVerdict: "tie",
			Base: `package main

func alpha() int {
	return 1
}
`,
			Ours: `package main

func alpha() int {
	return 10
}
`,
			Theirs: `package main

func alpha() int {
	return 20
}
`,
		},
		{
			Name:          "go_format_vs_logic",
			Description:   "One branch only reformats a function while the other changes its logic.",
			Language:      "go",
			Path:          "main.go",
			Tags:          []string{"merge", "go", "formatting"},
			ExpectVerdict: "ada_advantage",
			Base: `package main

func price(total int) int {
return total + 1
}
`,
			Ours: `package main

func price(total int) int {
	return total + 1
}
`,
			Theirs: `package main

func price(total int) int {
return total + 2
}
`,
		},
		{
			Name:          "go_independent_lines_same_symbol",
			Description:   "Both branches change different lines inside the same function body.",
			Language:      "go",
			Path:          "main.go",
			Tags:          []string{"merge", "go", "same-symbol"},
			ExpectVerdict: "git_advantage",
			Base: `package main

func compute(x int) int {
	offset := 1
	y := x + offset

	result := y * 2
	return result
}
`,
			Ours: `package main

func compute(x int) int {
	offset := 10
	y := x + offset

	result := y * 2
	return result
}
`,
			Theirs: `package main

func compute(x int) int {
	offset := 1
	y := x + offset

	result := y * 3
	return result
}
`,
		},
	}
}

func SelectScenarios(scenarios []Scenario, names []string, language string) []Scenario {
	if len(names) == 0 && language == "" {
		return append([]Scenario(nil), scenarios...)
	}
	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		nameSet[name] = struct{}{}
	}
	filtered := make([]Scenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		if language != "" && scenario.Language != language {
			continue
		}
		if len(nameSet) > 0 {
			if _, ok := nameSet[scenario.Name]; !ok {
				continue
			}
		}
		filtered = append(filtered, scenario)
	}
	return filtered
}
