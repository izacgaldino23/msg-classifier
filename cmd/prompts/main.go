// Command prompts is the batch validation runner for the Jev prompt examples: it
// reads a .json or .csv file of {flow, message, expected_result} rows and runs
// each through the exact production path (no database, no HTTP), printing ✓/✗ per
// row and a summary, and exiting non-zero when any of them misses — so a script
// or CI can gate on it. The same command regenerates the web harness seed SQL
// from the file (-sql), keeping the JSON the single source of the catalog.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"

	"msg-classifier/internal/app"
	"msg-classifier/internal/messages"
	"msg-classifier/internal/models"
	"msg-classifier/internal/services"
)

const defaultFile = "scripts/prompts/exemplos.json"

// row pairs a prompt with its evaluation so the report can show the flow (the
// EvaluationResult itself does not carry it).
type row struct {
	prompt models.JevPrompt
	result models.EvaluationResult
}

func main() {
	var (
		file    = flag.String("file", defaultFile, "prompts file (.json or .csv)")
		flow    = flag.String("flow", "", "run only this flow")
		workers = flag.Int("workers", 4, "parallel evaluations (1 serializes)")
		csvOut  = flag.String("csv", "", "write the results to this csv path")
		sqlOut  = flag.String("sql", "", "write the seed SQL generated from the file and exit")
	)
	flag.Parse()

	prompts, err := services.LoadPromptsFile(*file)
	if err != nil {
		fail(err)
	}
	if *flow != "" {
		filtered := make([]models.JevPrompt, 0, len(prompts))
		for _, prompt := range prompts {
			if prompt.Flow == *flow {
				filtered = append(filtered, prompt)
			}
		}
		if len(filtered) == 0 {
			fail(errors.New(messages.PromptsFileEmpty(*flow, *file)))
		}
		prompts = filtered
	}

	if *sqlOut != "" {
		if err := os.WriteFile(*sqlOut, []byte(services.SeedSQL(prompts)), 0o644); err != nil {
			fail(fmt.Errorf("failed to write seed sql: %w", err))
		}
		fmt.Println(messages.PromptSQLWritten(len(prompts), *sqlOut))
		return
	}

	// The file is data, not database rows; positions keep the CSV ids useful.
	for i := range prompts {
		prompts[i].ID = uint(i + 1)
	}

	rows := run(app.NewEvaluator(), prompts, *workers)

	if *csvOut != "" {
		csvFile, err := os.Create(*csvOut)
		if err != nil {
			fail(fmt.Errorf("failed to create csv: %w", err))
		}
		defer csvFile.Close()
		if err := services.WriteResultsCSV(csvFile, resultsOf(rows)); err != nil {
			fail(err)
		}
		fmt.Println(messages.PromptCSVWritten(*csvOut))
	}

	ok := 0
	for _, r := range rows {
		if r.result.Match {
			ok++
			fmt.Println(messages.PromptRunOk(r.prompt.Flow, r.result.ObtainedResult))
		} else {
			fmt.Println(messages.PromptRunMismatch(r.prompt.Flow, r.result.ExpectedResult, r.result.ObtainedResult))
		}
	}
	failures := len(rows) - ok
	fmt.Println(messages.PromptRunSummary(ok, len(rows), failures))
	if failures > 0 {
		os.Exit(1)
	}
}

// run evaluates the prompts through the shared database-free evaluator with a
// worker pool; results come back in file order. The evaluator itself stays
// sequential and pure — the parallelism is a runner concern.
func run(eval *services.PromptEvaluator, prompts []models.JevPrompt, workers int) []row {
	if workers < 1 {
		workers = 1
	}
	rows := make([]row, len(prompts))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i := range prompts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			rows[i] = row{prompt: prompts[i], result: eval.Evaluate([]models.JevPrompt{prompts[i]})[0]}
		}(i)
	}
	wg.Wait()
	return rows
}

func resultsOf(rows []row) []models.EvaluationResult {
	results := make([]models.EvaluationResult, len(rows))
	for i, r := range rows {
		results[i] = r.result
	}
	return results
}

// fail prints an error to stderr and exits 1.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}