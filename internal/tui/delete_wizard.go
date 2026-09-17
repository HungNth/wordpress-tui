package tui

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"wptui/internal/deprovision"
)

// BuildDeleteSelectionForm constructs a MultiSelect form allowing selection of candidates.
func BuildDeleteSelectionForm(candidates []deprovision.Candidate, selectedSlugs *[]string) *huh.Form {
	opts := make([]huh.Option[string], 0, len(candidates))
	for _, c := range candidates {
		opts = append(opts, huh.NewOption(c.Slug, c.Slug))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select Websites to Delete").
				Description("Choose one or more websites to decommission permanently.").
				Options(opts...).
				Value(selectedSlugs),
		),
	).WithTheme(CustomTheme())
}

// PromptDeleteSelection prompts user to pick candidate websites to delete.
func PromptDeleteSelection(candidates []deprovision.Candidate) ([]deprovision.Candidate, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	var selectedSlugs []string
	form := BuildDeleteSelectionForm(candidates, &selectedSlugs)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}

	if len(selectedSlugs) == 0 {
		return nil, nil
	}

	selectedMap := make(map[string]bool)
	for _, s := range selectedSlugs {
		selectedMap[s] = true
	}

	var chosen []deprovision.Candidate
	for _, c := range candidates {
		if selectedMap[c.Slug] {
			chosen = append(chosen, c)
		}
	}

	return chosen, nil
}

// BuildDeleteConfirmMultiForm builds the confirmation form with formatted summary table.
func BuildDeleteConfirmMultiForm(selected []deprovision.Candidate, confirmed *bool) *huh.Form {
	var sb strings.Builder
	sb.WriteString("WARNING: The following resources will be permanently deleted!\n\n")
	for i, c := range selected {
		dbDisplay := c.DetectedDB
		if dbDisplay == "" {
			dbDisplay = "unknown (will not be dropped)"
		}
		sb.WriteString(fmt.Sprintf("%d. Directory: %s\n   Path:      %s\n   Database:  %s\n", i+1, c.Slug, c.Path, dbDisplay))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Permanently delete %d website(s)?", len(selected))).
				Description(sb.String()).
				Value(confirmed).
				Affirmative("Yes, permanently delete").
				Negative("No, cancel"),
		),
	).WithTheme(CustomTheme())
}

// PromptDeleteConfirmMulti displays the multi-site confirmation dialog defaulting to No.
func PromptDeleteConfirmMulti(selected []deprovision.Candidate) (bool, error) {
	if len(selected) == 0 {
		return false, nil
	}

	var confirmed bool
	form := BuildDeleteConfirmMultiForm(selected, &confirmed)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, nil
		}
		return false, err
	}

	return confirmed, nil
}

// PromptDeleteConfirm displays a single candidate confirmation prompt defaulting to No.
func PromptDeleteConfirm(c deprovision.Candidate) (bool, error) {
	return PromptDeleteConfirmMulti([]deprovision.Candidate{c})
}

// PrintDeleteResult outputs a structured summary of the de-provisioning outcome.
func PrintDeleteResult(res deprovision.Result) {
	fmt.Printf("\n=== De-provisioning Result for %q ===\n", res.Candidate.Slug)
	fmt.Printf("Path: %s\n", res.Candidate.Path)

	// Herd
	if res.HerdDone {
		fmt.Println("  - Herd TLS:       unsecured")
	} else if res.HerdErr != nil {
		fmt.Printf("  - Herd TLS:       warning: %v\n", res.HerdErr)
	} else {
		fmt.Println("  - Herd TLS:       not applicable")
	}

	// Database
	if res.DBDone {
		fmt.Printf("  - Database:       deleted (%s)\n", res.Candidate.DetectedDB)
	} else if res.DBErr != nil {
		fmt.Printf("  - Database:       error: %v\n", res.DBErr)
	} else if res.Candidate.DetectedDB != "" {
		fmt.Printf("  - Database:       skipped\n")
	} else {
		fmt.Println("  - Database:       unknown (not deleted)")
	}

	// Directory
	if res.DirDone {
		fmt.Println("  - Directory:      deleted")
	} else if res.DirErr != nil {
		fmt.Printf("  - Directory:      error: %v\n", res.DirErr)
	}
	fmt.Println()
}

// PrintDeleteSummary prints a formatted aggregate report for all de-provisioning results.
func PrintDeleteSummary(results []deprovision.Result) {
	if len(results) == 0 {
		return
	}

	fmt.Println("\n================ De-provisioning Summary ================")
	var totalDeletedDirs int
	var totalDroppedDBs int
	var totalUnsecuredHerd int

	for i, res := range results {
		fmt.Printf("[%d/%d] %s (%s)\n", i+1, len(results), res.Candidate.Slug, res.Candidate.Path)

		// Herd status
		if res.HerdDone {
			fmt.Println("    • Herd TLS:  unsecured")
			totalUnsecuredHerd++
		} else if res.HerdErr != nil {
			fmt.Printf("    • Herd TLS:  warning: %v\n", res.HerdErr)
		} else {
			fmt.Println("    • Herd TLS:  not applicable")
		}

		// DB status
		if res.DBDone {
			fmt.Printf("    • Database:  deleted (%s)\n", res.Candidate.DetectedDB)
			totalDroppedDBs++
		} else if res.DBErr != nil {
			fmt.Printf("    • Database:  error: %v\n", res.DBErr)
		} else if res.Candidate.DetectedDB != "" {
			fmt.Println("    • Database:  skipped")
		} else {
			fmt.Println("    • Database:  unknown (not deleted)")
		}

		// Directory status
		if res.DirDone {
			fmt.Println("    • Directory: deleted")
			totalDeletedDirs++
		} else if res.DirErr != nil {
			fmt.Printf("    • Directory: error: %v\n", res.DirErr)
		}
	}

	fmt.Printf("\nTotals: %d/%d directories deleted, %d databases dropped, %d TLS certs unsecured.\n",
		totalDeletedDirs, len(results), totalDroppedDBs, totalUnsecuredHerd)
	fmt.Println("=========================================================")
}
