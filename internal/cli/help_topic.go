package cli

import "github.com/spf13/cobra"

// topicAnnotation marks a help topic so the help and grammar tests can
// find every topic in the tree without a hand-kept list.
const topicAnnotation = "topic"

// helpTopic builds a command that does no work and prints long, the same
// text whether run directly or through 'ynab help <topic>'.
func helpTopic(use, short, long string) *cobra.Command {
	return &cobra.Command{
		Use: use, Short: short, Long: long, Args: cobra.NoArgs,
		GroupID:     "topics",
		Annotations: map[string]string{topicAnnotation: "true"},
		RunE:        run(func(cmd *cobra.Command, _ []string) error { return cmd.Help() }),
	}
}
