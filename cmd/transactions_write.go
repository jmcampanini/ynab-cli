package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/names"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// Shared machinery for the mutating transaction commands: the write gate,
// name resolution against the plan, dry-run previews, and the bulk update
// with its batching.

// bulkBatch is the number of transactions per bulk update request, a CLI
// choice; the API states no limit.
const bulkBatch = 100

// creditCardPaymentsGroup is the API's internal group whose categories it
// ignores as a transaction's category.
const creditCardPaymentsGroup = "Credit Card Payments"

// writeSession is what every mutating command has in hand once the gate
// has passed: the client, the plan, and its accounts, plus the categories
// and payees once a lookup has loaded them.
type writeSession struct {
	accounts   []ynab.Account
	app        *app
	categories []ynab.CategoryGroup
	client     *ynab.Client
	dryRun     bool
	payees     []ynab.Payee
	plan       ynab.Plan
}

// openWrite loads the configuration, applies the write gate, and reads
// the plan and its accounts. Without --dry-run and without allow_writes
// it fails with exit status 3 before any request.
func (a *app) openWrite(cmd *cobra.Command, dryRun bool) (*writeSession, error) {
	loaded, client, err := a.connect(cmd)
	if err != nil {
		return nil, err
	}
	if !dryRun && !loaded.Config.AllowWrites {
		return nil, writesDisabled(loaded.Path)
	}

	ctx := cmd.Context()
	plan, err := selectedPlan(ctx, client, loaded.Config)
	if err != nil {
		return nil, err
	}
	accounts, err := client.Accounts(ctx, plan.ID)
	if err != nil {
		return nil, err
	}
	return &writeSession{accounts: accounts, app: a, client: client, dryRun: dryRun, plan: plan}, nil
}

// amount parses an amount typed in currency units at the plan's decimal
// precision. A bad value is a usage error.
func (s *writeSession) amount(text string) (ynab.Amount, error) {
	digits := 2
	if s.plan.CurrencyFormat != nil {
		digits = s.plan.CurrencyFormat.DecimalDigits
	}
	amount, err := ynab.ParseAmount(text, digits)
	if err != nil {
		return 0, usageError(err.Error())
	}
	return amount, nil
}

func (s *writeSession) account(query string) (ynab.Account, error) {
	return findAccount(query, s.accounts)
}

// accountByID returns the account, or a zero account when the ID is
// unknown to the plan.
func (s *writeSession) accountByID(id string) ynab.Account {
	for _, account := range s.accounts {
		if account.ID == id {
			return account
		}
	}
	return ynab.Account{}
}

// category resolves a category operand for a write. A category in the
// Credit Card Payments group is refused, since the API ignores it on a
// transaction and a card payment is a transfer.
func (s *writeSession) category(ctx context.Context, query string) (ynab.Category, error) {
	if s.categories == nil {
		groups, err := s.client.Categories(ctx, s.plan.ID)
		if err != nil {
			return ynab.Category{}, err
		}
		s.categories = append([]ynab.CategoryGroup{}, groups...)
	}

	category, group, err := findCategory(query, s.categories)
	if err != nil {
		return ynab.Category{}, err
	}
	if group.Internal && group.Name == creditCardPaymentsGroup {
		return ynab.Category{}, fmt.Errorf("category %q is a credit card payment category, which the API ignores on a transaction; record a card payment as a transfer with --transfer-to", query)
	}
	return category, nil
}

// categoryName returns the loaded name of a category, or the ID itself
// when the categories were never loaded or do not carry it.
func (s *writeSession) categoryName(id string) string {
	for _, group := range s.categories {
		for _, category := range group.Categories {
			if category.ID == id {
				return category.Name
			}
		}
	}
	return id
}

// payee resolves a --payee value: an ID or the exact name of an existing
// payee gives its ID; any other text is returned as a name for the API
// to create, unless it is shaped like an ID, since a mistyped ID must
// not become a payee. Several payees with that name is an error.
func (s *writeSession) payee(ctx context.Context, query string) (id, name string, err error) {
	if query == "" {
		return "", "", usageError("--payee needs a payee ID or name")
	}
	if s.payees == nil {
		payees, loadErr := s.client.Payees(ctx, s.plan.ID)
		if loadErr != nil {
			return "", "", loadErr
		}
		s.payees = append([]ynab.Payee{}, payees...)
	}

	payeeNames := make([]string, len(s.payees))
	for i, payee := range s.payees {
		if payee.ID == query {
			return payee.ID, "", nil
		}
		payeeNames[i] = payee.Name
	}
	matches := names.Match(query, payeeNames)
	switch len(matches) {
	case 0:
		if looksLikeID(query) {
			return "", "", fmt.Errorf("payee ID %q not found; a name would create a payee, but this looks like an ID", query)
		}
		return "", query, nil
	case 1:
		return s.payees[matches[0]].ID, "", nil
	}
	matched := make([]string, len(matches))
	for i, index := range matches {
		matched[i] = s.payees[index].ID
	}
	return "", "", fmt.Errorf("payee %q names %d payees; use an ID: %s", query, len(matches), strings.Join(matched, ", "))
}

// knownPayee reports whether the loaded payees carry the ID.
func (s *writeSession) knownPayee(id string) bool {
	for _, payee := range s.payees {
		if payee.ID == id {
			return true
		}
	}
	return false
}

// looksLikeID reports whether text has the API's UUID shape: 36
// characters with dashes after the 8th, 13th, 18th, and 23rd.
func looksLikeID(text string) bool {
	if len(text) != 36 {
		return false
	}
	for i, r := range text {
		dash := i == 8 || i == 13 || i == 18 || i == 23
		hex := r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'
		if dash && r != '-' || !dash && !hex {
			return false
		}
	}
	return true
}

// payeeName returns the display name of a payee ID: the transfer payee
// of an account is "Transfer : Account", others come from the loaded
// payees, and an unknown ID is returned as is.
func (s *writeSession) payeeName(id string) string {
	if account, ok := s.transferTarget(id); ok {
		return "Transfer : " + account.Name
	}
	for _, payee := range s.payees {
		if payee.ID == id {
			return payee.Name
		}
	}
	return id
}

// transferTarget returns the account whose transfer payee has the ID.
func (s *writeSession) transferTarget(payeeID string) (ynab.Account, bool) {
	for _, account := range s.accounts {
		if account.TransferPayeeID != nil && *account.TransferPayeeID == payeeID {
			return account, true
		}
	}
	return ynab.Account{}, false
}

// transferCategoryConflict rejects a category on a transfer between two
// plan accounts, where the API would ignore it.
func transferCategoryConflict(source, target ynab.Account) error {
	if source.OnPlan && target.OnPlan {
		return usageError(fmt.Sprintf("--category is not allowed on a transfer between the plan accounts %q and %q; the API ignores it", source.Name, target.Name))
	}
	return nil
}

// transactionsByID returns the transactions the IDs name, in that order,
// from one plan listing and a request per ID outside the API's default
// one-year window. An unknown ID fails.
func (s *writeSession) transactionsByID(ctx context.Context, ids []string) ([]ynab.Transaction, error) {
	listed, err := s.client.Transactions(ctx, s.plan.ID, ynab.TransactionFilter{})
	if err != nil {
		return nil, err
	}
	byID := make(map[string]ynab.Transaction, len(listed))
	for _, tx := range listed {
		byID[tx.ID] = tx
	}

	found := make([]ynab.Transaction, 0, len(ids))
	for _, id := range ids {
		tx, ok := byID[id]
		if !ok {
			if tx, err = s.client.Transaction(ctx, s.plan.ID, id); err != nil {
				return nil, fmt.Errorf("transaction %s: %w", id, err)
			}
		}
		found = append(found, tx)
	}
	return found, nil
}

// preview applies a change to a transaction as the API would, so a dry
// run prints what the real run would. An empty transaction previews a
// create, with the API's defaults for what the change leaves unset. On
// an existing split the API ignores the date, amount, and category, and
// a transfer between plan accounts carries no category.
func (s *writeSession) preview(tx ynab.Transaction, change ynab.SaveTransaction) ynab.Transaction {
	if tx.Cleared == "" {
		tx.Cleared = "uncleared"
	}
	if change.AccountID != "" {
		tx.AccountID = change.AccountID
		tx.AccountName = s.accountByID(change.AccountID).Name
	}
	splitParent := len(tx.Subtransactions) > 0
	if change.Date != "" && !splitParent {
		tx.Date = change.Date
	}
	if change.Milliunits != nil && !splitParent {
		tx.Amount = ynab.Amount(*change.Milliunits)
	}
	switch {
	case change.PayeeID != nil:
		id := *change.PayeeID
		name := s.payeeName(id)
		tx.PayeeID, tx.PayeeName = &id, &name
		tx.TransferAccountID, tx.TransferTransactionID = nil, nil
		if target, ok := s.transferTarget(id); ok {
			tx.TransferAccountID = &target.ID
			if target.OnPlan && s.accountByID(tx.AccountID).OnPlan {
				tx.CategoryID, tx.CategoryName = nil, nil
			}
		}
	case change.PayeeName != nil:
		tx.PayeeID, tx.PayeeName = nil, change.PayeeName
		tx.TransferAccountID, tx.TransferTransactionID = nil, nil
	}
	if change.CategoryID != nil && !splitParent {
		name := s.categoryName(*change.CategoryID)
		tx.CategoryID, tx.CategoryName = change.CategoryID, &name
	}
	if change.Memo != nil {
		tx.Memo = change.Memo
	}
	if change.Cleared != "" {
		tx.Cleared = change.Cleared
	}
	if change.Approved != nil {
		tx.Approved = *change.Approved
	}
	if change.FlagColor.Set {
		tx.FlagColor, tx.FlagName = change.FlagColor.Value, nil
	}
	if change.ImportID != "" {
		importID := change.ImportID
		tx.ImportID = &importID
	}
	if len(change.Subtransactions) > 0 {
		split := "Split"
		tx.CategoryID, tx.CategoryName = nil, &split
		tx.Subtransactions = make([]ynab.Subtransaction, len(change.Subtransactions))
		for i, line := range change.Subtransactions {
			preview := ynab.Subtransaction{Amount: ynab.Amount(line.Milliunits), Memo: line.Memo}
			if line.CategoryID != nil {
				name := s.categoryName(*line.CategoryID)
				preview.CategoryID, preview.CategoryName = line.CategoryID, &name
			}
			tx.Subtransactions[i] = preview
		}
	}
	return tx
}

// writeRecord is a transaction record printed by a mutating command,
// marked when it is a dry-run preview rather than a stored result.
type writeRecord struct {
	transactionRecord
	DryRun bool `json:"dry_run,omitempty"`
}

func (s *writeSession) records(transactions []ynab.Transaction) []writeRecord {
	accountNames := newAccountNames(s.accounts)
	records := make([]writeRecord, len(transactions))
	for i, tx := range transactions {
		records[i] = writeRecord{transactionRecord: newTransactionRecord(tx, accountNames), DryRun: s.dryRun}
	}
	return records
}

// writeCSVLines flattens splits as csvLines does, keeping the mark.
func writeCSVLines(records []writeRecord) []writeRecord {
	var rows []writeRecord
	for _, record := range records {
		for _, line := range csvLines([]transactionRecord{record.transactionRecord}) {
			rows = append(rows, writeRecord{transactionRecord: line, DryRun: record.DryRun})
		}
	}
	return rows
}

// summary is the closing line of a mutating command in human output:
// what it did, or on a dry run what it would have done. past and base
// are the verb's forms, such as "approved" and "approve".
func (s *writeSession) summary(past, base string, count int) string {
	var what string
	switch {
	case count == 0:
		what = "nothing to " + base
	case s.dryRun:
		what = fmt.Sprintf("would %s %s", base, transactionCount(count))
	default:
		what = fmt.Sprintf("%s %s", past, transactionCount(count))
	}
	if s.dryRun {
		return "dry run, nothing changed: " + what
	}
	return what
}

// printTransactions writes the records in the chosen format; human output
// is the register table followed by the summary line.
func (s *writeSession) printTransactions(out io.Writer, output outputFlags, transactions []ynab.Transaction, summary string) error {
	records := s.records(transactions)
	switch {
	case output.jsonl:
		return writeJSONL(out, records)
	case output.csv:
		return writeCSV(out, writeCSVLines(records))
	}

	plainRecords := make([]transactionRecord, len(records))
	for i, record := range records {
		plainRecords[i] = record.transactionRecord
	}
	if err := writeTable(out, transactionColumns, transactionRows(plainRecords, s.plan.CurrencyFormat)); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, summary)
	return err
}

// printTransaction writes one record as transactions get does, followed
// in human output by the summary line.
func (s *writeSession) printTransaction(out io.Writer, output outputFlags, tx ynab.Transaction, summary string) error {
	records := s.records([]ynab.Transaction{tx})
	if output.jsonl {
		return writeJSONL(out, records)
	}

	if err := writeTransactionFields(out, records[0].transactionRecord, s.plan.CurrencyFormat); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "\n%s\n", summary)
	return err
}

// bulkWrite is one change applied to many transactions. Either ids names
// them, or current holds them when the caller has already listed them.
// note, when set, follows the summary count, such as a payee created.
type bulkWrite struct {
	base    string
	change  ynab.SaveTransaction
	check   func(ynab.Transaction) error
	current []ynab.Transaction
	ids     []string
	note    string
	past    string
}

// summary phrases the closing line for this write.
func (w bulkWrite) summary(s *writeSession, count int) string {
	text := s.summary(w.past, w.base, count)
	if count > 0 && w.note != "" {
		text += " " + w.note
	}
	return text
}

// bulk applies the change to every transaction through bulk updates of
// bulkBatch each and prints the stored records. check, when set, vets
// each transaction's current state first, which costs the lookup; a dry
// run performs the lookup to print previews instead of updating. When a
// later batch fails, or the API returns fewer records than the batch
// named, the records already applied are printed before the error
// returns, and the error lists the IDs not changed.
func (s *writeSession) bulk(cmd *cobra.Command, output outputFlags, write bulkWrite) error {
	ctx := cmd.Context()
	ids := uniqueIDs(write.ids)
	current := write.current
	if current != nil {
		ids = transactionIDs(current)
	} else if s.dryRun || write.check != nil {
		var err error
		if current, err = s.transactionsByID(ctx, ids); err != nil {
			return err
		}
	}
	if write.check != nil {
		for _, tx := range current {
			if err := write.check(tx); err != nil {
				return err
			}
		}
	}

	out := cmd.OutOrStdout()
	if s.dryRun {
		previews := make([]ynab.Transaction, len(current))
		for i, tx := range current {
			previews[i] = s.preview(tx, write.change)
		}
		return s.printTransactions(out, output, previews, write.summary(s, len(previews)))
	}

	var applied []ynab.Transaction
	batches := (len(ids) + bulkBatch - 1) / bulkBatch
	for start := 0; start < len(ids); start += bulkBatch {
		batch := ids[start:min(start+bulkBatch, len(ids))]
		number := start/bulkBatch + 1
		if batches > 1 {
			// Progress is a diagnostic; losing it must not stop the update.
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "batch %d of %d: %s\n", number, batches, transactionCount(len(batch)))
		}
		changes := make([]ynab.SaveTransaction, len(batch))
		for i, id := range batch {
			changes[i] = write.change
			changes[i].ID = id
		}
		stored, err := s.client.UpdateTransactions(ctx, s.plan.ID, changes)
		if err == nil {
			err = missingRecords(batch, stored)
		}
		if err != nil {
			applied = append(applied, stored...)
			unsent := ids[start+len(stored):]
			err = fmt.Errorf("batch %d of %d failed after %s were applied: %w; %s not changed: %s", number, batches, transactionCount(len(applied)), err, transactionCount(len(unsent)), strings.Join(unsent, " "))
			if len(applied) == 0 {
				return err
			}
			return errors.Join(err, s.printTransactions(out, output, applied, write.summary(s, len(applied))))
		}
		s.warnIgnoredSplitFields(cmd, write.change, stored)
		applied = append(applied, stored...)
	}
	return s.printTransactions(out, output, applied, write.summary(s, len(applied)))
}

// missingRecords fails when the API's answer to a batch does not carry a
// record for every ID sent, so a short answer is never counted as done.
func missingRecords(batch []string, stored []ynab.Transaction) error {
	returned := make(map[string]bool, len(stored))
	for _, tx := range stored {
		returned[tx.ID] = true
	}
	var missing []string
	for _, id := range batch {
		if !returned[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("the API returned %d of %d records; missing: %s", len(stored), len(batch), strings.Join(missing, " "))
}

// warnIgnoredSplitFields says on stderr which stored records are splits
// whose date, amount, or category the API ignored, since a real run
// cannot know a transaction is a split before the update answers.
func (s *writeSession) warnIgnoredSplitFields(cmd *cobra.Command, change ynab.SaveTransaction, stored []ynab.Transaction) {
	if change.Date == "" && change.Milliunits == nil && change.CategoryID == nil {
		return
	}
	for _, tx := range stored {
		if len(tx.Subtransactions) > 0 {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "transaction %s is a split; the API ignored its date, amount, or category change\n", tx.ID)
		}
	}
}

// uniqueIDs drops repeated operands, keeping the first order.
func uniqueIDs(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	var unique []string
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	return unique
}

func transactionIDs(transactions []ynab.Transaction) []string {
	ids := make([]string, len(transactions))
	for i, tx := range transactions {
		ids[i] = tx.ID
	}
	return ids
}

// oneOfIDsOrAll rejects the operand and flag forms of a bulk command
// given together or not at all. all is the flag's name without dashes.
func oneOfIDsOrAll(args []string, all string, allSet bool) error {
	switch {
	case len(args) > 0 && allSet:
		return usageError(fmt.Sprintf("give transaction IDs or --%s, not both", all))
	case len(args) == 0 && !allSet:
		return usageError(fmt.Sprintf("give at least one transaction ID or --%s", all))
	}
	return nil
}

// transactionFields are the field flags create and update share, as
// typed; resolve turns the ones given into a change.
type transactionFields struct {
	account, amount, category, cleared, date, flag, memo, payee, transferTo string
	approved                                                                bool
}

// transactionFieldFlags are the flag names transactionFields binds.
var transactionFieldFlags = []string{"account", "date", "amount", "payee", "transfer-to", "category", "memo", "cleared", "approved", "flag"}

func (f *transactionFields) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.account, "account", "", "Account ID or exact name")
	cmd.Flags().StringVar(&f.date, "date", "", "Date: YYYY-MM-DD, today, yesterday")
	cmd.Flags().StringVar(&f.amount, "amount", "", "Amount in currency units, outflows negative")
	cmd.Flags().StringVar(&f.payee, "payee", "", "Payee ID or name; an unknown name creates the payee")
	cmd.Flags().StringVar(&f.transferTo, "transfer-to", "", "Make it a transfer to this account ID or exact name")
	cmd.Flags().StringVar(&f.category, "category", "", "Category ID, exact name, or \"Group: Name\"")
	cmd.Flags().StringVar(&f.memo, "memo", "", "Memo text; an empty value clears it")
	bindEnumFlag(cmd, &f.cleared, "cleared", "state", "Cleared state: uncleared, cleared, reconciled", clearedStates...)
	cmd.Flags().BoolVar(&f.approved, "approved", false, "Mark approved; --approved=false marks unapproved")
	bindEnumFlag(cmd, &f.flag, "flag", "color", "Flag color, or none to remove the flag", flagColors...)
	cmd.MarkFlagsMutuallyExclusive("payee", "transfer-to")
}

// given reports whether any field flag was set.
func (f *transactionFields) given(cmd *cobra.Command) bool {
	for _, name := range transactionFieldFlags {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

// resolve builds the change from the flags that were given, resolving
// names through the session. transfer is the target account when
// --transfer-to was given or --payee named an account's transfer payee,
// so the caller can apply the on-plan check.
func (f *transactionFields) resolve(cmd *cobra.Command, s *writeSession) (change ynab.SaveTransaction, transfer *ynab.Account, err error) {
	ctx := cmd.Context()
	flags := cmd.Flags()
	if flags.Changed("account") {
		account, err := s.account(f.account)
		if err != nil {
			return change, nil, err
		}
		change.AccountID = account.ID
	}
	if flags.Changed("date") {
		if change.Date, err = s.app.date(f.date); err != nil {
			return change, nil, err
		}
	}
	if flags.Changed("amount") {
		amount, err := s.amount(f.amount)
		if err != nil {
			return change, nil, err
		}
		change.SetAmount(amount)
	}
	if flags.Changed("payee") {
		id, name, err := s.payee(ctx, f.payee)
		if err != nil {
			return change, nil, err
		}
		if id != "" {
			change.PayeeID = &id
		} else {
			change.PayeeName = &name
		}
		if target, ok := s.transferTarget(id); ok {
			transfer = &target
		}
	}
	if flags.Changed("transfer-to") {
		target, err := s.account(f.transferTo)
		if err != nil {
			return change, nil, err
		}
		if target.TransferPayeeID == nil {
			return change, nil, fmt.Errorf("account %q has no transfer payee, so the API cannot transfer to it", target.Name)
		}
		change.PayeeID = target.TransferPayeeID
		transfer = &target
	}
	if flags.Changed("category") {
		category, err := s.category(ctx, f.category)
		if err != nil {
			return change, nil, err
		}
		change.CategoryID = &category.ID
	}
	if flags.Changed("memo") {
		memo := f.memo
		change.Memo = &memo
	}
	if flags.Changed("cleared") {
		change.Cleared = f.cleared
	}
	if flags.Changed("approved") {
		approved := f.approved
		change.Approved = &approved
	}
	if flags.Changed("flag") {
		change.FlagColor = ynab.SomeValue(f.flag)
		if f.flag == "none" {
			change.FlagColor = ynab.NullValue[string]()
		}
	}
	return change, transfer, nil
}
