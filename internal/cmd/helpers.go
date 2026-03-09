package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/bstn/monarch-cli/internal/app"
	"github.com/bstn/monarch-cli/internal/config"
	"github.com/bstn/monarch-cli/internal/output"
)

func prompt(rt *app.Runtime, label string) (string, error) {
	if _, err := fmt.Fprintf(rt.Stderr, "%s: ", label); err != nil {
		return "", err
	}
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func promptPassword(rt *app.Runtime, label string) (string, error) {
	if _, err := fmt.Fprintf(rt.Stderr, "%s: ", label); err != nil {
		return "", err
	}
	bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	_, _ = fmt.Fprintln(rt.Stderr)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func decodeJSONMap(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseDate(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return "", fmt.Errorf("invalid date %q: expected YYYY-MM-DD", value)
	}
	return parsed.Format("2006-01-02"), nil
}

func currentMonthRange(now time.Time) (string, string) {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, -1)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func last30DaysRange(now time.Time) (string, string) {
	end := now
	start := now.AddDate(0, 0, -30)
	return start.Format("2006-01-02"), end.Format("2006-01-02")
}

func last31DaysStart(now time.Time) string {
	return now.AddDate(0, 0, -31).Format("2006-01-02")
}

func mustClient(ctx context.Context, rt *app.Runtime) (*config.Profile, any, string, error) {
	client, profile, source, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return nil, nil, "", err
	}
	return profile, client, source, nil
}

func printMaybeRaw[T any](rt *app.Runtime, raw any, normalized T, table output.Table) error {
	if rt.Raw {
		return rt.PrintJSON(raw)
	}
	return rt.Print(normalized, table)
}
