package cmd

import (
	"context"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// Shell completion of the operands and flags that take an account,
// category, category group, or payee name. Each completer reads the plan
// once and offers the names the matching list command shows by default.
// A completer never fails the shell: when the configuration, the plan,
// or the request fails, it offers nothing.

// completer is a Cobra completion function for an operand or a flag.
type completer = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)

// nameLister returns the names a completer offers from the plan.
type nameLister func(ctx context.Context, client *ynab.Client, planID string) ([]string, error)

// complete builds a completer that lists names with one request after
// resolving the plan, keeping those that start with the typed prefix
// ignoring case.
func (a *app) complete(list nameLister) completer {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		loaded, client, err := a.connect(cmd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		plan, err := selectedPlan(cmd.Context(), client, loaded.Config)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		names, err := list(cmd.Context(), client, plan.ID)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var matches []string
		for _, name := range names {
			if strings.HasPrefix(strings.ToLower(name), strings.ToLower(toComplete)) {
				matches = append(matches, name)
			}
		}
		return matches, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeAccounts offers open accounts.
func (a *app) completeAccounts() completer {
	return a.complete(func(ctx context.Context, client *ynab.Client, planID string) ([]string, error) {
		accounts, err := client.Accounts(ctx, planID)
		if err != nil {
			return nil, err
		}
		var names []string
		for _, account := range accounts {
			if !account.Closed {
				names = append(names, account.Name)
			}
		}
		return names, nil
	})
}

// completeCategories offers the categories 'categories list' shows,
// minus the API's own, which no operand accepts. A name shared with any
// other category, hidden ones included, is offered as "Group: Name",
// since the bare name would fail to resolve.
func (a *app) completeCategories() completer {
	return a.complete(func(ctx context.Context, client *ynab.Client, planID string) ([]string, error) {
		groups, err := client.Categories(ctx, planID)
		if err != nil {
			return nil, err
		}
		return completionNames(groups), nil
	})
}

// completeSources offers the categories plus the ready-to-assign operand
// of --from and --to.
func (a *app) completeSources() completer {
	categories := a.completeCategories()
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		names, directive := categories(cmd, args, toComplete)
		if strings.HasPrefix(readyToAssignOperand, strings.ToLower(toComplete)) {
			names = append(names, readyToAssignOperand)
		}
		return names, directive
	}
}

// completionNames returns the name of each shown category, qualified by its
// group when any category, hidden ones included, shares the name.
func completionNames(groups []ynab.CategoryGroup) []string {
	shared := map[string]int{}
	for _, group := range groups {
		for _, category := range group.Categories {
			shared[strings.ToLower(category.Name)]++
		}
	}
	var names []string
	for _, group := range groups {
		for _, category := range group.Categories {
			switch {
			case category.Internal || category.Hidden || group.Hidden:
				continue
			case shared[strings.ToLower(category.Name)] > 1:
				names = append(names, group.Name+": "+category.Name)
			default:
				names = append(names, category.Name)
			}
		}
	}
	return names
}

// completeCategoryGroups offers the groups 'category-groups list' shows.
func (a *app) completeCategoryGroups() completer {
	return a.complete(func(ctx context.Context, client *ynab.Client, planID string) ([]string, error) {
		groups, err := client.Categories(ctx, planID)
		if err != nil {
			return nil, err
		}
		var names []string
		for _, group := range groups {
			if !group.Hidden {
				names = append(names, group.Name)
			}
		}
		return names, nil
	})
}

// completePayees offers every payee.
func (a *app) completePayees() completer {
	return a.complete(func(ctx context.Context, client *ynab.Client, planID string) ([]string, error) {
		payees, err := client.Payees(ctx, planID)
		if err != nil {
			return nil, err
		}
		names := make([]string, len(payees))
		for i, payee := range payees {
			names[i] = payee.Name
		}
		return names, nil
	})
}

// firstOperand limits a completer to the first operand, for commands
// whose later operands are amounts or new names.
func firstOperand(complete completer) completer {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return complete(cmd, args, toComplete)
	}
}

// completeFlags registers one completer for each named flag.
func completeFlags(cmd *cobra.Command, complete completer, flags ...string) {
	for _, flag := range flags {
		// The flags are declared by the same constructor, so a missing
		// one is a programming error.
		if err := cmd.RegisterFlagCompletionFunc(flag, complete); err != nil {
			panic(err)
		}
	}
}

// completeTransactionFields registers completers for the name-taking
// field flags of transactions create and update.
func completeTransactionFields(cmd *cobra.Command, a *app) {
	completeFlags(cmd, a.completeAccounts(), "account", "transfer-to")
	completeFlags(cmd, a.completeCategories(), "category")
	completeFlags(cmd, a.completePayees(), "payee")
}
