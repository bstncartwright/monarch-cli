package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"github.com/bstn/monarch-cli/internal/app"
	"github.com/bstn/monarch-cli/internal/config"
	"github.com/bstn/monarch-cli/internal/secrets"
)

type RootFlags struct {
	Format  string `help:"Output format: json|table" default:"json" enum:"json,table"`
	Raw     bool   `help:"Return raw GraphQL response shape for first-class commands"`
	Profile string `help:"Stored profile name" default:"default"`
	Token   string `help:"Use a Monarch token directly for this invocation" env:"MONARCH_TOKEN"`
	Config  string `help:"Config file path override"`
}

type CLI struct {
	RootFlags `embed:""`

	Auth         AuthCmd         `cmd:"" help:"Authentication and profile management"`
	Me           MeCmd           `cmd:"" help:"Current user profile"`
	Accounts     AccountsCmd     `cmd:"" help:"Accounts, balances, holdings, and history"`
	Transactions TransactionsCmd `cmd:"" help:"Transactions and summaries"`
	Budgets      BudgetsCmd      `cmd:"" help:"Budget data"`
	Cashflow     CashflowCmd     `cmd:"" help:"Cashflow summaries"`
	Recurring    RecurringCmd    `cmd:"" help:"Recurring items"`
	GraphQL      GraphQLCmd      `cmd:"" name:"graphql" aliases:"gql" help:"Raw GraphQL queries"`
	Version      VersionCmd      `cmd:"" help:"Print version"`
}

type ExecuteOptions struct {
	Stdout        io.Writer
	Stderr        io.Writer
	ConfigPath    string
	BaseURL       string
	HTTPClient    *http.Client
	ConfigManager *config.Manager
	SecretStore   secrets.Store
}

func Execute(args []string) error {
	return ExecuteWithOptions(args, ExecuteOptions{})
}

func ExecuteWithOptions(args []string, opts ExecuteOptions) error {
	cli := CLI{}
	parser, err := kong.New(&cli,
		kong.Name("monarch"),
		kong.Description("JSON-first CLI for accessing Monarch Money data.\n\nIf your Monarch account uses Google or Apple sign-in, create a direct password in Monarch's Security settings before using `monarch auth login`."),
		kong.UsageOnError(),
	)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		args = []string{"--help"}
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		return err
	}
	rt, err := app.New(app.Options{
		Stdout:        opts.Stdout,
		Stderr:        opts.Stderr,
		Format:        cli.Format,
		Raw:           cli.Raw,
		Profile:       cli.Profile,
		TokenOverride: cli.Token,
		ConfigPath:    firstNonEmpty(cli.Config, opts.ConfigPath),
		BaseURL:       opts.BaseURL,
		HTTPClient:    opts.HTTPClient,
		ConfigManager: opts.ConfigManager,
		SecretStore:   opts.SecretStore,
	})
	if err != nil {
		return err
	}
	kctx.Bind(rt)
	kctx.Bind(&cli.RootFlags)
	if err := kctx.Run(); err != nil {
		return err
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

func ExitCode(err error) int {
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	if err != nil {
		return 1
	}
	return 0
}

type VersionCmd struct{}

func (VersionCmd) Run() error {
	_, err := fmt.Fprintln(os.Stdout, "monarch dev")
	return err
}
