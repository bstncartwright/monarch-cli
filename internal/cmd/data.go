package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bstn/monarch-cli/internal/app"
	"github.com/bstn/monarch-cli/internal/monarch"
	"github.com/bstn/monarch-cli/internal/output"
)

type MeCmd struct {
	Get MeGetCmd `cmd:"" help:"Get the current user profile"`
}

type MeGetCmd struct{}

func (c *MeGetCmd) Run(rt *app.Runtime) error {
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.GetMe(ctx)
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	return printMaybeRaw(rt, resp, resp.Data.Me, output.Table{
		Headers: []string{"ID", "NAME", "EMAIL", "TIMEZONE"},
		Rows:    [][]string{{resp.Data.Me.ID, resp.Data.Me.Name, resp.Data.Me.Email, resp.Data.Me.Timezone}},
	})
}

type AccountsCmd struct {
	List     AccountsListCmd     `cmd:"" help:"List accounts"`
	Balances AccountsBalancesCmd `cmd:"" help:"Get recent balances for all accounts"`
	Holdings AccountsHoldingsCmd `cmd:"" help:"Get holdings for an account"`
	History  AccountsHistoryCmd  `cmd:"" help:"Get account history and recent transactions"`
}

type AccountsListCmd struct{}

func (c *AccountsListCmd) Run(rt *app.Runtime) error {
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.GetAccounts(ctx)
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	rows := make([][]string, 0, len(resp.Data.Accounts))
	for _, account := range resp.Data.Accounts {
		rows = append(rows, []string{
			account.ID,
			account.DisplayName,
			floatString(account.CurrentBalance),
			account.Type.Display,
			account.Subtype.Display,
		})
	}
	return printMaybeRaw(rt, resp, resp.Data.Accounts, output.Table{
		Headers: []string{"ID", "NAME", "BALANCE", "TYPE", "SUBTYPE"},
		Rows:    rows,
	})
}

type AccountsBalancesCmd struct {
	StartDate string `name:"start-date" help:"Start date for recent balances (YYYY-MM-DD)"`
}

func (c *AccountsBalancesCmd) Run(rt *app.Runtime) error {
	start := strings.TrimSpace(c.StartDate)
	if start == "" {
		start = last31DaysStart(time.Now())
	}
	parsed, err := parseDate(start)
	if err != nil {
		return err
	}
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.GetRecentBalances(ctx, parsed)
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	out := make([]monarch.AccountBalances, 0, len(resp.Data.Accounts))
	rows := make([][]string, 0, len(resp.Data.Accounts))
	for _, account := range resp.Data.Accounts {
		out = append(out, monarch.AccountBalances{
			ID:             account.ID,
			DisplayName:    account.DisplayName,
			StartDate:      parsed,
			RecentBalances: account.RecentBalances,
		})
		rows = append(rows, []string{
			account.ID,
			account.DisplayName,
			fmt.Sprintf("%d", len(account.RecentBalances)),
			parsed,
		})
	}
	return printMaybeRaw(rt, resp, out, output.Table{
		Headers: []string{"ID", "NAME", "POINTS", "START_DATE"},
		Rows:    rows,
	})
}

type AccountsHoldingsCmd struct {
	AccountID string `arg:"" required:"" help:"Monarch account id"`
}

func (c *AccountsHoldingsCmd) Run(rt *app.Runtime) error {
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.GetAccountHoldings(ctx, c.AccountID, time.Now())
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	holdings := make([]monarch.Holding, 0, len(resp.Data.Portfolio.AggregateHoldings.Edges))
	rows := make([][]string, 0, len(resp.Data.Portfolio.AggregateHoldings.Edges))
	for _, edge := range resp.Data.Portfolio.AggregateHoldings.Edges {
		holdings = append(holdings, edge.Node)
		rows = append(rows, []string{
			edge.Node.Security.Ticker,
			edge.Node.Security.Name,
			float64String(edge.Node.Quantity),
			float64String(edge.Node.TotalValue),
		})
	}
	return printMaybeRaw(rt, resp, holdings, output.Table{
		Headers: []string{"TICKER", "NAME", "QUANTITY", "TOTAL_VALUE"},
		Rows:    rows,
	})
}

type AccountsHistoryCmd struct {
	AccountID string `arg:"" required:"" help:"Monarch account id"`
}

func (c *AccountsHistoryCmd) Run(rt *app.Runtime) error {
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.GetAccountHistory(ctx, c.AccountID)
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	normalized := map[string]any{
		"account":             resp.Data.Account,
		"snapshots":           resp.Data.Snapshots,
		"recent_transactions": resp.Data.Transactions,
	}
	return printMaybeRaw(rt, resp, normalized, output.Table{})
}

type TransactionsCmd struct {
	List    TransactionsListCmd    `cmd:"" help:"List transactions"`
	Get     TransactionsGetCmd     `cmd:"" help:"Get a single transaction"`
	Summary TransactionsSummaryCmd `cmd:"" help:"Get transaction summary aggregates"`
}

type TransactionsListCmd struct {
	Limit     int    `help:"Result limit" default:"100"`
	Offset    int    `help:"Result offset" default:"0"`
	StartDate string `name:"start-date" help:"Start date filter (YYYY-MM-DD)"`
	EndDate   string `name:"end-date" help:"End date filter (YYYY-MM-DD)"`
	Search    string `help:"Search query"`
}

func (c *TransactionsListCmd) Run(rt *app.Runtime) error {
	start := strings.TrimSpace(c.StartDate)
	end := strings.TrimSpace(c.EndDate)
	if start == "" && end == "" {
		start, end = last30DaysRange(time.Now())
	} else {
		var err error
		start, err = parseDate(start)
		if err != nil {
			return err
		}
		end, err = parseDate(end)
		if err != nil {
			return err
		}
	}
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.ListTransactions(ctx, monarch.TransactionsListParams{
		Limit:     c.Limit,
		Offset:    c.Offset,
		OrderBy:   "inverse_date",
		StartDate: start,
		EndDate:   end,
		Search:    c.Search,
	})
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	rows := make([][]string, 0, len(resp.Data.AllTransactions.Results))
	for _, tx := range resp.Data.AllTransactions.Results {
		rows = append(rows, []string{
			tx.ID,
			tx.Date,
			float64String(tx.Amount),
			tx.Merchant.Name,
			tx.Account.DisplayName,
		})
	}
	normalized := map[string]any{
		"total_count": resp.Data.AllTransactions.TotalCount,
		"results":     resp.Data.AllTransactions.Results,
		"start_date":  start,
		"end_date":    end,
	}
	return printMaybeRaw(rt, resp, normalized, output.Table{
		Headers: []string{"ID", "DATE", "AMOUNT", "MERCHANT", "ACCOUNT"},
		Rows:    rows,
	})
}

type TransactionsGetCmd struct {
	TransactionID string `arg:"" required:"" help:"Transaction id"`
}

func (c *TransactionsGetCmd) Run(rt *app.Runtime) error {
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.GetTransaction(ctx, c.TransactionID)
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	tx := resp.Data.GetTransaction
	return printMaybeRaw(rt, resp, tx, output.Table{
		Headers: []string{"ID", "DATE", "AMOUNT", "MERCHANT", "CATEGORY"},
		Rows:    [][]string{{tx.ID, tx.Date, float64String(tx.Amount), tx.Merchant.Name, tx.Category.Name}},
	})
}

type TransactionsSummaryCmd struct {
	StartDate string `name:"start-date" help:"Start date filter (YYYY-MM-DD)"`
	EndDate   string `name:"end-date" help:"End date filter (YYYY-MM-DD)"`
}

func (c *TransactionsSummaryCmd) Run(rt *app.Runtime) error {
	start := strings.TrimSpace(c.StartDate)
	end := strings.TrimSpace(c.EndDate)
	if start == "" && end == "" {
		start, end = last30DaysRange(time.Now())
	} else {
		var err error
		start, err = parseDate(start)
		if err != nil {
			return err
		}
		end, err = parseDate(end)
		if err != nil {
			return err
		}
	}
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.GetTransactionsSummary(ctx, monarch.DateRange{StartDate: start, EndDate: end})
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	var summary monarch.TransactionsSummary
	if len(resp.Data.Aggregates) > 0 {
		summary = resp.Data.Aggregates[0].Summary
	}
	normalized := map[string]any{
		"start_date": start,
		"end_date":   end,
		"summary":    summary,
	}
	return printMaybeRaw(rt, resp, normalized, output.Table{
		Headers: []string{"INCOME", "EXPENSE", "SAVINGS", "SAVINGS_RATE"},
		Rows: [][]string{{
			floatString(summary.SumIncome),
			floatString(summary.SumExpense),
			floatString(summary.Savings),
			floatString(summary.SavingsRate),
		}},
	})
}

type BudgetsCmd struct {
	Get BudgetsGetCmd `cmd:"" help:"Get budget data"`
}

type BudgetsGetCmd struct {
	StartDate string `name:"start-date" help:"Start month (YYYY-MM-DD)"`
	EndDate   string `name:"end-date" help:"End month (YYYY-MM-DD)"`
}

func (c *BudgetsGetCmd) Run(rt *app.Runtime) error {
	start := strings.TrimSpace(c.StartDate)
	end := strings.TrimSpace(c.EndDate)
	if start == "" && end == "" {
		start, end = currentMonthRange(time.Now())
	} else {
		var err error
		start, err = parseDate(start)
		if err != nil {
			return err
		}
		end, err = parseDate(end)
		if err != nil {
			return err
		}
	}
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.GetBudgets(ctx, start, end)
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	rows := [][]string{}
	for _, month := range resp.Data.BudgetData.TotalsByMonth {
		rows = append(rows, []string{
			month.Month,
			floatString(month.TotalIncome.PlannedAmount),
			floatString(month.TotalExpenses.PlannedAmount),
			floatString(month.TotalExpenses.ActualAmount),
		})
	}
	return printMaybeRaw(rt, resp, resp.Data, output.Table{
		Headers: []string{"MONTH", "PLANNED_INCOME", "PLANNED_EXPENSES", "ACTUAL_EXPENSES"},
		Rows:    rows,
	})
}

type CashflowCmd struct {
	Get CashflowGetCmd `cmd:"" help:"Get cashflow aggregates"`
}

type CashflowGetCmd struct {
	StartDate string `name:"start-date" help:"Start date (YYYY-MM-DD)"`
	EndDate   string `name:"end-date" help:"End date (YYYY-MM-DD)"`
}

func (c *CashflowGetCmd) Run(rt *app.Runtime) error {
	start := strings.TrimSpace(c.StartDate)
	end := strings.TrimSpace(c.EndDate)
	if start == "" && end == "" {
		start, end = currentMonthRange(time.Now())
	} else {
		var err error
		start, err = parseDate(start)
		if err != nil {
			return err
		}
		end, err = parseDate(end)
		if err != nil {
			return err
		}
	}
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.GetCashflow(ctx, start, end)
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	var summary monarch.TransactionsSummary
	if len(resp.Data.Summary) > 0 {
		summary = resp.Data.Summary[0].Summary
	}
	normalized := map[string]any{
		"start_date":        start,
		"end_date":          end,
		"summary":           summary,
		"by_category_group": resp.Data.ByCategoryGroup,
		"by_category":       resp.Data.ByCategory,
		"by_merchant":       resp.Data.ByMerchant,
	}
	return printMaybeRaw(rt, resp, normalized, output.Table{
		Headers: []string{"INCOME", "EXPENSE", "SAVINGS", "SAVINGS_RATE"},
		Rows:    [][]string{{floatString(summary.SumIncome), floatString(summary.SumExpense), floatString(summary.Savings), floatString(summary.SavingsRate)}},
	})
}

type RecurringCmd struct {
	List RecurringListCmd `cmd:"" help:"List recurring items"`
}

type RecurringListCmd struct {
	StartDate string `name:"start-date" help:"Start date (YYYY-MM-DD)"`
	EndDate   string `name:"end-date" help:"End date (YYYY-MM-DD)"`
}

func (c *RecurringListCmd) Run(rt *app.Runtime) error {
	start := strings.TrimSpace(c.StartDate)
	end := strings.TrimSpace(c.EndDate)
	if start == "" && end == "" {
		start, end = currentMonthRange(time.Now())
	} else {
		var err error
		start, err = parseDate(start)
		if err != nil {
			return err
		}
		end, err = parseDate(end)
		if err != nil {
			return err
		}
	}
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.GetRecurringItems(ctx, start, end)
	if err != nil {
		return err
	}
	if err := monarch.ValidateGraphQLResponse(resp); err != nil {
		return err
	}
	rows := make([][]string, 0, len(resp.Data.RecurringTransactionItems))
	for _, item := range resp.Data.RecurringTransactionItems {
		rows = append(rows, []string{
			item.Date,
			item.Stream.Merchant.Name,
			item.Account.DisplayName,
			float64String(item.Amount),
		})
	}
	return printMaybeRaw(rt, resp, resp.Data.RecurringTransactionItems, output.Table{
		Headers: []string{"DATE", "MERCHANT", "ACCOUNT", "AMOUNT"},
		Rows:    rows,
	})
}

type GraphQLCmd struct {
	Query GraphQLQueryCmd `cmd:"" help:"Run a raw GraphQL query"`
}

type GraphQLQueryCmd struct {
	File      string `help:"Read query text from a file"`
	Stdin     bool   `name:"stdin" help:"Read query text from stdin"`
	Variables string `help:"Variables JSON object"`
	QueryText string `arg:"" optional:"" help:"GraphQL query text"`
}

func (c *GraphQLQueryCmd) Run(rt *app.Runtime) error {
	query := strings.TrimSpace(c.QueryText)
	if c.File != "" {
		data, err := os.ReadFile(c.File)
		if err != nil {
			return err
		}
		query = strings.TrimSpace(string(data))
	}
	if c.Stdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		query = strings.TrimSpace(string(data))
	}
	if query == "" {
		return fmt.Errorf("graphql query text is required")
	}
	variables, err := decodeJSONMap(c.Variables)
	if err != nil {
		return fmt.Errorf("parse variables: %w", err)
	}
	ctx := context.Background()
	client, _, _, err := rt.ClientFromConfig(ctx)
	if err != nil {
		return err
	}
	resp, err := client.RawQuery(ctx, query, variables)
	if err != nil {
		return err
	}
	return rt.PrintJSON(resp)
}

func rtNow(_ *app.Runtime) time.Time {
	return time.Now()
}

func floatString(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.2f", *value)
}

func float64String(value float64) string {
	return fmt.Sprintf("%.2f", value)
}
