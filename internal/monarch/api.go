package monarch

import (
	"context"
	"fmt"
	"time"
)

const (
	meQuery = `
query Me {
  me {
    id
    name
    displayName
    email
    hasMfaOn
    hasPassword
    householdRole
    createdAt
    timezone
  }
}`

	accountsQuery = `
query GetAccounts {
  accounts {
    id
    displayName
    currentBalance
    displayLastUpdatedAt
    includeInNetWorth
    hideFromList
    hideTransactionsFromReports
    isAsset
    isHidden
    isManual
    mask
    transactionsCount
    holdingsCount
    dataProvider
    logoUrl
    type { name display }
    subtype { name display }
    institution { id name url }
  }
}`

	recentBalancesQuery = `
query GetRecentBalances($startDate: Date!) {
  accounts {
    id
    displayName
    recentBalances(startDate: $startDate)
  }
}`

	accountHoldingsQuery = `
query GetHoldings($input: PortfolioInput) {
  portfolio(input: $input) {
    aggregateHoldings {
      edges {
        node {
          id
          quantity
          basis
          totalValue
          securityPriceChangeDollars
          securityPriceChangePercent
          lastSyncedAt
          holdings {
            id
            type
            typeDisplay
            name
            ticker
            closingPrice
            isManual
          }
          security {
            id
            name
            type
            ticker
            typeDisplay
            currentPrice
            oneDayChangePercent
            oneDayChangeDollars
          }
        }
      }
    }
  }
}`

	accountHistoryQuery = `
query AccountHistory($id: UUID!, $filters: TransactionFilterInput) {
  account(id: $id) {
    id
    displayName
    currentBalance
    displayLastUpdatedAt
    mask
    isAsset
    isManual
    type { name display }
    subtype { name display }
    institution { id name url }
  }
  transactions: allTransactions(filters: $filters) {
    totalCount
    results(limit: 20, orderBy: inverse_date) {
      id
      date
      amount
      pending
      notes
      merchant { id name }
      category { id name }
    }
  }
  snapshots: snapshotsForAccount(accountId: $id) {
    date
    signedBalance
  }
}`

	transactionsListQuery = `
query Transactions($filters: TransactionFilterInput, $limit: Int!, $offset: Int!, $orderBy: TransactionOrdering) {
  allTransactions(filters: $filters) {
    totalCount
    results(limit: $limit, offset: $offset, orderBy: $orderBy) {
      id
      date
      amount
      pending
      notes
      isRecurring
      needsReview
      hideFromReports
      merchant { id name }
      category { id name }
      account { id displayName }
      tags { id name color }
    }
  }
}`

	transactionGetQuery = `
query GetTransaction($id: UUID!) {
  getTransaction(id: $id) {
    id
    date
    originalDate
    amount
    pending
    notes
    isRecurring
    needsReview
    reviewStatus
    hideFromReports
    dataProviderDescription
    merchant { id name logoUrl }
    category { id name }
    account { id displayName }
    tags { id name color }
    splitTransactions {
      id
      amount
      notes
      category { id name }
      merchant { id name }
    }
  }
}`

	transactionSummaryQuery = `
query TransactionSummary($filters: TransactionFilterInput) {
  aggregates(filters: $filters, fillEmptyValues: true) {
    summary {
      sumIncome
      sumExpense
      savings
      savingsRate
    }
  }
}`

	budgetsQuery = `
query Budgets($startDate: Date!, $endDate: Date!) {
  budgetSystem
  budgetData(startMonth: $startDate, endMonth: $endDate) {
    monthlyAmountsByCategory {
      category { id }
      monthlyAmounts {
        month
        plannedCashFlowAmount
        actualAmount
        remainingAmount
      }
    }
    totalsByMonth {
      month
      totalIncome { actualAmount plannedAmount remainingAmount }
      totalExpenses { actualAmount plannedAmount remainingAmount }
      totalFixedExpenses { actualAmount plannedAmount remainingAmount }
      totalFlexibleExpenses { actualAmount plannedAmount remainingAmount }
      totalNonMonthlyExpenses { actualAmount plannedAmount remainingAmount }
    }
  }
  categoryGroups {
    id
    name
    type
    categories {
      id
      name
      icon
      budgetVariability
    }
  }
  goalsV2 {
    id
    name
    archivedAt
    completedAt
  }
}`

	cashflowQuery = `
query Cashflow($filters: TransactionFilterInput) {
  byCategory: aggregates(filters: $filters, groupBy: ["category"]) {
    groupBy {
      category {
        id
        name
        group { id type name }
      }
    }
    summary {
      sum
      sumIncome
      sumExpense
    }
  }
  byCategoryGroup: aggregates(filters: $filters, groupBy: ["categoryGroup"]) {
    groupBy {
      categoryGroup {
        id
        name
        type
      }
    }
    summary {
      sum
      sumIncome
      sumExpense
    }
  }
  byMerchant: aggregates(filters: $filters, groupBy: ["merchant"]) {
    groupBy {
      merchant {
        id
        name
        logoUrl
      }
    }
    summary {
      sumIncome
      sumExpense
    }
  }
  summary: aggregates(filters: $filters, fillEmptyValues: true) {
    summary {
      sum
      sumIncome
      sumExpense
      savings
      savingsRate
    }
  }
}`

	recurringItemsQuery = `
query RecurringItems($startDate: Date!, $endDate: Date!, $filters: RecurringTransactionFilter) {
  recurringTransactionItems(startDate: $startDate, endDate: $endDate, filters: $filters) {
    stream {
      id
      frequency
      amount
      isApproximate
      merchant { id name logoUrl }
    }
    date
    isPast
    transactionId
    amount
    amountDiff
    category { id name }
    account { id displayName logoUrl }
  }
}`
)

type User struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	Email         string `json:"email"`
	HasMFAOn      bool   `json:"hasMfaOn"`
	HasPassword   bool   `json:"hasPassword"`
	HouseholdRole string `json:"householdRole"`
	CreatedAt     string `json:"createdAt"`
	Timezone      string `json:"timezone"`
}

type Account struct {
	ID                         string      `json:"id"`
	DisplayName                string      `json:"displayName"`
	CurrentBalance             *float64    `json:"currentBalance"`
	DisplayLastUpdatedAt       string      `json:"displayLastUpdatedAt"`
	IncludeInNetWorth          bool        `json:"includeInNetWorth"`
	HideFromList               bool        `json:"hideFromList"`
	HideTransactionsFromReport bool        `json:"hideTransactionsFromReports"`
	IsAsset                    bool        `json:"isAsset"`
	IsHidden                   bool        `json:"isHidden"`
	IsManual                   bool        `json:"isManual"`
	Mask                       string      `json:"mask"`
	TransactionsCount          int         `json:"transactionsCount"`
	HoldingsCount              int         `json:"holdingsCount"`
	DataProvider               string      `json:"dataProvider"`
	LogoURL                    string      `json:"logoUrl"`
	Type                       AccountType `json:"type"`
	Subtype                    AccountType `json:"subtype"`
	Institution                Institution `json:"institution"`
}

type AccountType struct {
	Name    string `json:"name"`
	Display string `json:"display"`
}

type Institution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type AccountBalances struct {
	ID             string    `json:"id"`
	DisplayName    string    `json:"displayName"`
	StartDate      string    `json:"startDate"`
	RecentBalances []float64 `json:"recentBalances"`
}

type HoldingsResponse struct {
	AggregateHoldings struct {
		Edges []struct {
			Node Holding `json:"node"`
		} `json:"edges"`
	} `json:"aggregateHoldings"`
}

type Holding struct {
	ID                         string          `json:"id"`
	Quantity                   float64         `json:"quantity"`
	Basis                      float64         `json:"basis"`
	TotalValue                 float64         `json:"totalValue"`
	SecurityPriceChangeDollars *float64        `json:"securityPriceChangeDollars"`
	SecurityPriceChangePercent *float64        `json:"securityPriceChangePercent"`
	LastSyncedAt               string          `json:"lastSyncedAt"`
	Holdings                   []HoldingLot    `json:"holdings"`
	Security                   HoldingSecurity `json:"security"`
}

type HoldingLot struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	TypeDisplay  string   `json:"typeDisplay"`
	Name         string   `json:"name"`
	Ticker       string   `json:"ticker"`
	ClosingPrice *float64 `json:"closingPrice"`
	IsManual     bool     `json:"isManual"`
}

type HoldingSecurity struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Type                string   `json:"type"`
	Ticker              string   `json:"ticker"`
	TypeDisplay         string   `json:"typeDisplay"`
	CurrentPrice        *float64 `json:"currentPrice"`
	OneDayChangePercent *float64 `json:"oneDayChangePercent"`
	OneDayChangeDollars *float64 `json:"oneDayChangeDollars"`
}

type Snapshot struct {
	Date          string   `json:"date"`
	SignedBalance *float64 `json:"signedBalance"`
}

type Merchant struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	LogoURL string `json:"logoUrl"`
}

type CategoryGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Category struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Group CategoryGroup `json:"group"`
}

type TransactionTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type TransactionAccount struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	LogoURL     string `json:"logoUrl"`
}

type Transaction struct {
	ID                      string               `json:"id"`
	Date                    string               `json:"date"`
	OriginalDate            string               `json:"originalDate"`
	Amount                  float64              `json:"amount"`
	Pending                 bool                 `json:"pending"`
	Notes                   string               `json:"notes"`
	IsRecurring             bool                 `json:"isRecurring"`
	NeedsReview             bool                 `json:"needsReview"`
	HideFromReports         bool                 `json:"hideFromReports"`
	ReviewStatus            string               `json:"reviewStatus"`
	DataProviderDescription string               `json:"dataProviderDescription"`
	Merchant                Merchant             `json:"merchant"`
	Category                Category             `json:"category"`
	Account                 TransactionAccount   `json:"account"`
	Tags                    []TransactionTag     `json:"tags"`
	SplitTransactions       []TransactionSummary `json:"splitTransactions"`
}

type TransactionSummary struct {
	ID       string   `json:"id"`
	Amount   float64  `json:"amount"`
	Notes    string   `json:"notes"`
	Category Category `json:"category"`
	Merchant Merchant `json:"merchant"`
}

type TransactionsPage struct {
	TotalCount int           `json:"totalCount"`
	Results    []Transaction `json:"results"`
}

type TransactionsSummary struct {
	SumIncome   *float64 `json:"sumIncome"`
	SumExpense  *float64 `json:"sumExpense"`
	Savings     *float64 `json:"savings"`
	SavingsRate *float64 `json:"savingsRate"`
}

type BudgetMonthlyAmount struct {
	Month                 string   `json:"month"`
	PlannedCashFlowAmount *float64 `json:"plannedCashFlowAmount"`
	ActualAmount          *float64 `json:"actualAmount"`
	RemainingAmount       *float64 `json:"remainingAmount"`
}

type BudgetCategoryAmounts struct {
	Category struct {
		ID string `json:"id"`
	} `json:"category"`
	MonthlyAmounts []BudgetMonthlyAmount `json:"monthlyAmounts"`
}

type BudgetTotals struct {
	ActualAmount    *float64 `json:"actualAmount"`
	PlannedAmount   *float64 `json:"plannedAmount"`
	RemainingAmount *float64 `json:"remainingAmount"`
}

type BudgetMonthTotals struct {
	Month                string       `json:"month"`
	TotalIncome          BudgetTotals `json:"totalIncome"`
	TotalExpenses        BudgetTotals `json:"totalExpenses"`
	TotalFixedExpenses   BudgetTotals `json:"totalFixedExpenses"`
	TotalFlexibleExpense BudgetTotals `json:"totalFlexibleExpenses"`
	TotalNonMonthly      BudgetTotals `json:"totalNonMonthlyExpenses"`
}

type BudgetCategory struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Icon              string `json:"icon"`
	BudgetVariability string `json:"budgetVariability"`
}

type BudgetCategoryGroup struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Type       string           `json:"type"`
	Categories []BudgetCategory `json:"categories"`
}

type Goal struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ArchivedAt  string `json:"archivedAt"`
	CompletedAt string `json:"completedAt"`
}

type BudgetsPayload struct {
	BudgetSystem string `json:"budgetSystem"`
	BudgetData   struct {
		MonthlyAmountsByCategory []BudgetCategoryAmounts `json:"monthlyAmountsByCategory"`
		TotalsByMonth            []BudgetMonthTotals     `json:"totalsByMonth"`
	} `json:"budgetData"`
	CategoryGroups []BudgetCategoryGroup `json:"categoryGroups"`
	Goals          []Goal                `json:"goalsV2"`
}

type Aggregate struct {
	GroupBy struct {
		Category      Category      `json:"category"`
		CategoryGroup CategoryGroup `json:"categoryGroup"`
		Merchant      Merchant      `json:"merchant"`
	} `json:"groupBy"`
	Summary TransactionsSummary `json:"summary"`
}

type CashflowPayload struct {
	ByCategory      []Aggregate `json:"byCategory"`
	ByCategoryGroup []Aggregate `json:"byCategoryGroup"`
	ByMerchant      []Aggregate `json:"byMerchant"`
	Summary         []Aggregate `json:"summary"`
}

type RecurringItem struct {
	Stream struct {
		ID            string   `json:"id"`
		Frequency     string   `json:"frequency"`
		Amount        *float64 `json:"amount"`
		IsApproximate bool     `json:"isApproximate"`
		Merchant      Merchant `json:"merchant"`
	} `json:"stream"`
	Date          string             `json:"date"`
	IsPast        bool               `json:"isPast"`
	TransactionID string             `json:"transactionId"`
	Amount        float64            `json:"amount"`
	AmountDiff    *float64           `json:"amountDiff"`
	Category      Category           `json:"category"`
	Account       TransactionAccount `json:"account"`
}

type DateRange struct {
	StartDate string
	EndDate   string
}

type TransactionsListParams struct {
	Limit     int
	Offset    int
	OrderBy   string
	StartDate string
	EndDate   string
	Search    string
}

type TransactionsFilters struct {
	StartDate string `json:"startDate,omitempty"`
	EndDate   string `json:"endDate,omitempty"`
	Search    string `json:"search,omitempty"`
}

func (c *Client) GetMe(ctx context.Context) (*GraphQLResponse[struct {
	Me User `json:"me"`
}], error) {
	var resp GraphQLResponse[struct {
		Me User `json:"me"`
	}]
	if err := c.Query(ctx, meQuery, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetAccounts(ctx context.Context) (*GraphQLResponse[struct {
	Accounts []Account `json:"accounts"`
}], error) {
	var resp GraphQLResponse[struct {
		Accounts []Account `json:"accounts"`
	}]
	if err := c.Query(ctx, accountsQuery, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetRecentBalances(ctx context.Context, startDate string) (*GraphQLResponse[struct {
	Accounts []struct {
		ID             string    `json:"id"`
		DisplayName    string    `json:"displayName"`
		RecentBalances []float64 `json:"recentBalances"`
	} `json:"accounts"`
}], error) {
	var resp GraphQLResponse[struct {
		Accounts []struct {
			ID             string    `json:"id"`
			DisplayName    string    `json:"displayName"`
			RecentBalances []float64 `json:"recentBalances"`
		} `json:"accounts"`
	}]
	if err := c.Query(ctx, recentBalancesQuery, map[string]any{"startDate": startDate}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetAccountHoldings(ctx context.Context, accountID string, today time.Time) (*GraphQLResponse[struct {
	Portfolio HoldingsResponse `json:"portfolio"`
}], error) {
	day := today.Format("2006-01-02")
	var resp GraphQLResponse[struct {
		Portfolio HoldingsResponse `json:"portfolio"`
	}]
	if err := c.Query(ctx, accountHoldingsQuery, map[string]any{
		"input": map[string]any{
			"accountIds":            []string{accountID},
			"startDate":             day,
			"endDate":               day,
			"includeHiddenHoldings": true,
		},
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetAccountHistory(ctx context.Context, accountID string) (*GraphQLResponse[struct {
	Account      Account          `json:"account"`
	Transactions TransactionsPage `json:"transactions"`
	Snapshots    []Snapshot       `json:"snapshots"`
}], error) {
	var resp GraphQLResponse[struct {
		Account      Account          `json:"account"`
		Transactions TransactionsPage `json:"transactions"`
		Snapshots    []Snapshot       `json:"snapshots"`
	}]
	if err := c.Query(ctx, accountHistoryQuery, map[string]any{
		"id": accountID,
		"filters": map[string]any{
			"accounts": []string{accountID},
		},
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListTransactions(ctx context.Context, params TransactionsListParams) (*GraphQLResponse[struct {
	AllTransactions TransactionsPage `json:"allTransactions"`
}], error) {
	var resp GraphQLResponse[struct {
		AllTransactions TransactionsPage `json:"allTransactions"`
	}]
	filters := map[string]any{}
	if params.StartDate != "" {
		filters["startDate"] = params.StartDate
	}
	if params.EndDate != "" {
		filters["endDate"] = params.EndDate
	}
	if params.Search != "" {
		filters["search"] = params.Search
	}
	if err := c.Query(ctx, transactionsListQuery, map[string]any{
		"filters": filters,
		"limit":   params.Limit,
		"offset":  params.Offset,
		"orderBy": params.OrderBy,
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetTransaction(ctx context.Context, id string) (*GraphQLResponse[struct {
	GetTransaction Transaction `json:"getTransaction"`
}], error) {
	var resp GraphQLResponse[struct {
		GetTransaction Transaction `json:"getTransaction"`
	}]
	if err := c.Query(ctx, transactionGetQuery, map[string]any{"id": id}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetTransactionsSummary(ctx context.Context, rangeFilter DateRange) (*GraphQLResponse[struct {
	Aggregates []Aggregate `json:"aggregates"`
}], error) {
	var resp GraphQLResponse[struct {
		Aggregates []Aggregate `json:"aggregates"`
	}]
	variables := map[string]any{"filters": map[string]any{}}
	if rangeFilter.StartDate != "" {
		variables["filters"] = map[string]any{
			"startDate": rangeFilter.StartDate,
			"endDate":   rangeFilter.EndDate,
		}
	}
	if err := c.Query(ctx, transactionSummaryQuery, variables, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetBudgets(ctx context.Context, startDate, endDate string) (*GraphQLResponse[BudgetsPayload], error) {
	var resp GraphQLResponse[BudgetsPayload]
	if err := c.Query(ctx, budgetsQuery, map[string]any{
		"startDate": startDate,
		"endDate":   endDate,
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetCashflow(ctx context.Context, startDate, endDate string) (*GraphQLResponse[CashflowPayload], error) {
	var resp GraphQLResponse[CashflowPayload]
	if err := c.Query(ctx, cashflowQuery, map[string]any{
		"filters": map[string]any{
			"search":     "",
			"categories": []string{},
			"accounts":   []string{},
			"tags":       []string{},
			"startDate":  startDate,
			"endDate":    endDate,
		},
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetRecurringItems(ctx context.Context, startDate, endDate string) (*GraphQLResponse[struct {
	RecurringTransactionItems []RecurringItem `json:"recurringTransactionItems"`
}], error) {
	var resp GraphQLResponse[struct {
		RecurringTransactionItems []RecurringItem `json:"recurringTransactionItems"`
	}]
	if err := c.Query(ctx, recurringItemsQuery, map[string]any{
		"startDate": startDate,
		"endDate":   endDate,
		"filters":   map[string]any{},
	}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) RawQuery(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	var resp map[string]any
	if err := c.Query(ctx, query, variables, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func ValidateGraphQLResponse[T any](resp *GraphQLResponse[T]) error {
	if resp == nil {
		return fmt.Errorf("missing graphql response")
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", resp.Errors[0].Message)
	}
	return nil
}
