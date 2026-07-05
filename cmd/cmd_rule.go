package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/strobotti/linkquisition"
)

var ruleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Manage browser match rules",
	Long:  "List, add, or remove URL match rules that determine which browser opens a URL.",
	RunE:  runRuleList,
}

var ruleListCmd = &cobra.Command{
	Use:   "list [browser]",
	Short: "List match rules",
	Long: `List URL match rules for all browsers, or for a specific browser.
The browser name is matched case-insensitively (partial match supported).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRuleList,
}

var ruleAddCmd = &cobra.Command{
	Use:   "add <browser> <type> <value>",
	Short: "Add a match rule to a browser",
	Long: `Add a URL match rule to a browser.

Types:
  site     Match the full hostname (e.g. "www.example.com")
  domain   Match the domain (e.g. "example.com" matches www.example.com too)
  regex    Match the full URL against a regular expression

Examples:
  linkquisition rule add firefox site www.facebook.com
  linkquisition rule add "Microsoft Edge" domain office.com
  linkquisition rule add chrome regex ".*\.google\.com/maps.*"`,
	Args: cobra.ExactArgs(3), //nolint:mnd
	RunE: runRuleAdd,
}

var ruleRemoveCmd = &cobra.Command{
	Use:   "remove <browser> <index>",
	Short: "Remove a match rule from a browser",
	Long: `Remove a match rule by its index (as shown in "rule list").
The index is 1-based.

Example:
  linkquisition rule remove firefox 1`,
	Args: cobra.ExactArgs(2), //nolint:mnd
	RunE: runRuleRemove,
}

func initRuleCmd() {
	ruleCmd.AddCommand(ruleListCmd)
	ruleCmd.AddCommand(ruleAddCmd)
	ruleCmd.AddCommand(ruleRemoveCmd)
}

func runRuleList(_ *cobra.Command, args []string) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	if len(args) == 1 {
		return listRulesForBrowser(settings, args[0])
	}

	return listAllRules(settings)
}

func listAllRules(settings *linkquisition.Settings) error {
	anyRules := false

	for _, b := range settings.Browsers {
		if len(b.Matches) == 0 {
			continue
		}

		anyRules = true
		fmt.Printf("%s:\n", b.Name)

		for i, m := range b.Matches {
			fmt.Printf("  %d. %-8s %s\n", i+1, m.Type, m.Value)
		}

		fmt.Println()
	}

	if !anyRules {
		fmt.Println("No match rules configured.")
	}

	return nil
}

func listRulesForBrowser(settings *linkquisition.Settings, browserName string) error {
	idx, err := findBrowserByName(settings, browserName)
	if err != nil {
		return err
	}

	b := settings.Browsers[idx]
	if len(b.Matches) == 0 {
		fmt.Printf("%s: no match rules configured.\n", b.Name)
		return nil
	}

	fmt.Printf("%s:\n", b.Name)

	for i, m := range b.Matches {
		fmt.Printf("  %d. %-8s %s\n", i+1, m.Type, m.Value)
	}

	return nil
}

func runRuleAdd(_ *cobra.Command, args []string) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	browserName := args[0]
	matchType := strings.ToLower(args[1])
	matchValue := args[2]

	// Validate match type
	validTypes := []string{
		linkquisition.BrowserMatchTypeSite,
		linkquisition.BrowserMatchTypeDomain,
		linkquisition.BrowserMatchTypeRegex,
	}

	valid := false
	for _, t := range validTypes {
		if matchType == t {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid rule type %q, must be one of: %s", matchType, strings.Join(validTypes, ", "))
	}

	// Validate regex compiles
	if matchType == linkquisition.BrowserMatchTypeRegex {
		if _, err := regexp.Compile(matchValue); err != nil {
			return fmt.Errorf("invalid regex pattern %q: %w", matchValue, err)
		}
	}

	// Find the browser
	idx, err := findBrowserByName(settings, browserName)
	if err != nil {
		return err
	}

	// Add the rule
	settings.Browsers[idx].Matches = append(settings.Browsers[idx].Matches, linkquisition.BrowserMatch{
		Type:  matchType,
		Value: matchValue,
	})

	if err := settingsService.WriteSettings(settings); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	fmt.Printf("Added: %s %s (to %s)\n", matchType, matchValue, settings.Browsers[idx].Name)

	return nil
}

func runRuleRemove(_ *cobra.Command, args []string) error {
	settingsService := newSettingsServiceForCLI()
	settings := settingsService.GetSettings()

	browserName := args[0]

	ruleIndex, err := strconv.Atoi(args[1])
	if err != nil || ruleIndex < 1 {
		return fmt.Errorf("invalid rule index %q: must be a positive integer (see \"rule list\")", args[1])
	}

	// Find the browser
	idx, err := findBrowserByName(settings, browserName)
	if err != nil {
		return err
	}

	b := &settings.Browsers[idx]
	if ruleIndex > len(b.Matches) {
		return fmt.Errorf(
			"rule index %d out of range: %s has %d rule(s)",
			ruleIndex, b.Name, len(b.Matches),
		)
	}

	// Remove the rule (1-based index)
	removed := b.Matches[ruleIndex-1]
	b.Matches = append(b.Matches[:ruleIndex-1], b.Matches[ruleIndex:]...)

	if err := settingsService.WriteSettings(settings); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	fmt.Printf("Removed: %s %s (from %s)\n", removed.Type, removed.Value, b.Name)

	return nil
}

// findBrowserByName finds a browser by case-insensitive partial name match.
// Returns the index into settings.Browsers, or an error if no match or ambiguous.
func findBrowserByName(settings *linkquisition.Settings, name string) (int, error) {
	nameLower := strings.ToLower(name)
	var matches []int

	for i, b := range settings.Browsers {
		if strings.EqualFold(b.Name, name) {
			// Exact match — return immediately
			return i, nil
		}

		if strings.Contains(strings.ToLower(b.Name), nameLower) {
			matches = append(matches, i)
		}
	}

	switch len(matches) {
	case 0:
		available := make([]string, 0, len(settings.Browsers))
		for _, b := range settings.Browsers {
			available = append(available, b.Name)
		}

		return -1, fmt.Errorf("no browser matching %q\nConfigured browsers: %s", name, strings.Join(available, ", "))
	case 1:
		return matches[0], nil
	default:
		ambiguous := make([]string, 0, len(matches))
		for _, idx := range matches {
			ambiguous = append(ambiguous, settings.Browsers[idx].Name)
		}

		return -1, fmt.Errorf("ambiguous browser name %q, matches: %s", name, strings.Join(ambiguous, ", "))
	}
}
