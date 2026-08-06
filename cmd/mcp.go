package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/ollama/ollama/mcp"
)

// mcpPaths resolves both files the MCP commands work with. They are resolved
// together so a command can never read the configuration from one place and the
// approvals from another.
type mcpPaths struct {
	config    string
	approvals string
}

func resolveMCPPaths() (mcpPaths, error) {
	config, err := mcp.ConfigPath()
	if err != nil {
		return mcpPaths{}, err
	}
	approvals, err := mcp.ApprovalsPath()
	if err != nil {
		return mcpPaths{}, err
	}
	return mcpPaths{config: config, approvals: approvals}, nil
}

func loadMCPState() (*mcp.Config, *mcp.Approvals, mcpPaths, error) {
	paths, err := resolveMCPPaths()
	if err != nil {
		return nil, nil, mcpPaths{}, err
	}
	cfg, err := mcp.Load(paths.config)
	if err != nil {
		return nil, nil, paths, err
	}
	approvals, err := mcp.LoadApprovals(paths.approvals)
	if err != nil {
		return nil, nil, paths, err
	}
	return cfg, approvals, paths, nil
}

// MCPCmd builds the `ollama mcp` command group.
func MCPCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP servers",
		Long: "Manage Model Context Protocol servers.\n\n" +
			"Servers are defined in mcp.json and must be approved before Ollama will run\n" +
			"them. An approval records the exact command line, so a server whose command\n" +
			"changes afterwards must be approved again.",
	}

	root.AddCommand(
		mcpListCmd(),
		mcpAddCmd(),
		mcpRemoveCmd(),
		mcpEnableDisableCmd("enable", false, "Enable an MCP server"),
		mcpEnableDisableCmd("disable", true, "Disable an MCP server"),
		mcpApproveCmd(),
		mcpRevokeCmd(),
		mcpLoginCmd(),
		mcpLogoutCmd(),
	)
	return root
}

func mcpListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured MCP servers",
		Long: "List configured MCP servers and whether Ollama may run them.\n\n" +
			"This does not start or contact any server.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, approvals, paths, err := loadMCPState()
			if err != nil {
				return err
			}

			names := cfg.Names()
			if len(names) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No MCP servers configured in %s\n", paths.config)
				return nil
			}
			problems := cfg.Problems()

			writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(writer, "NAME\tSTATUS\tRUNS")
			for _, name := range names {
				spec, _ := cfg.Get(name)
				fmt.Fprintf(writer, "%s\t%s\t%s\n", name, mcpStatusWord(spec, problems[name], approvals), spec.Summary())
			}
			if err := writer.Flush(); err != nil {
				return err
			}

			if unapproved := mcpUnapproved(cfg, problems, approvals); len(unapproved) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\nNot approved to run: %s\n", strings.Join(unapproved, ", "))
				fmt.Fprintf(cmd.OutOrStdout(), "Approve with: ollama mcp approve <name>\n")
			}
			return nil
		},
	}
}

// mcpStatusWord describes one server in a single word, in the order the user
// cares about: a broken entry first, then their own switch, then approval.
func mcpStatusWord(spec *mcp.ServerSpec, problem error, approvals *mcp.Approvals) string {
	switch {
	case problem != nil:
		return "invalid"
	case spec.Disabled:
		return "disabled"
	case approvals.Allows(spec):
		return "approved"
	case approvals.Entries[spec.Name].Fingerprint != "":
		// Approved once, but not for what it now says it would run.
		return "changed"
	default:
		return "not approved"
	}
}

func mcpUnapproved(cfg *mcp.Config, problems map[string]error, approvals *mcp.Approvals) []string {
	var names []string
	for _, name := range cfg.Names() {
		spec, _ := cfg.Get(name)
		if problems[name] == nil && !spec.Disabled && !approvals.Allows(spec) {
			names = append(names, name)
		}
	}
	return names
}

func mcpAddCmd() *cobra.Command {
	var url string
	var envPairs []string
	var headerPairs []string
	var noApprove bool

	cmd := &cobra.Command{
		Use:   "add NAME [COMMAND [ARG...]]",
		Short: "Add an MCP server",
		Long: "Add an MCP server, either a local command or a remote URL.\n\n" +
			"A server added here is approved to run, because the command line came from\n" +
			"you. Use --no-approve to add it without approving. Credentials must be given\n" +
			"as ${env:NAME} references rather than literal values.\n\n" +
			"Examples:\n" +
			"  ollama mcp add files uvx mcp-server-files\n" +
			"  ollama mcp add issues --url https://mcp.example.com/v1 --header 'Authorization=${env:EXAMPLE_TOKEN}'",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			rest := args[1:]

			spec := &mcp.ServerSpec{}
			switch {
			case url != "":
				if len(rest) > 0 {
					return errors.New("give a command or a --url, not both")
				}
				spec.Type = mcp.TransportHTTP
				spec.URL = url
			case len(rest) > 0:
				spec.Type = mcp.TransportStdio
				spec.Command = rest[0]
				spec.Args = rest[1:]
			default:
				return errors.New("give a command to run, or a --url")
			}

			var err error
			if spec.Env, err = parseKeyValues(envPairs, "--env"); err != nil {
				return err
			}
			if spec.Headers, err = parseKeyValues(headerPairs, "--header"); err != nil {
				return err
			}

			cfg, approvals, paths, err := loadMCPState()
			if err != nil {
				return err
			}
			if _, exists := cfg.Get(name); exists {
				return fmt.Errorf("an MCP server named %q already exists; remove it first", name)
			}

			cfg.Set(name, spec)
			// Validate through the same path the runtime uses, so a server that
			// could never start is refused here rather than at the next launch.
			if problem := cfg.Problems()[name]; problem != nil {
				return fmt.Errorf("%s: %w", name, problem)
			}

			if err := cfg.Save(paths.config); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Added %s\n  runs: %s\n", name, spec.Summary())

			if noApprove {
				fmt.Fprintf(out, "\nNot approved. Ollama will not run it until you run:\n  ollama mcp approve %s\n", name)
				return nil
			}

			stored, _ := cfg.Get(name)
			approvals.Approve(stored, time.Now())
			if err := approvals.Save(paths.approvals); err != nil {
				return err
			}
			fmt.Fprintf(out, "  approved to run\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "URL of a remote MCP server")
	cmd.Flags().StringArrayVar(&envPairs, "env", nil, "environment variable for a local server, as KEY=VALUE")
	cmd.Flags().StringArrayVar(&headerPairs, "header", nil, "HTTP header for a remote server, as NAME=VALUE")
	cmd.Flags().BoolVar(&noApprove, "no-approve", false, "add the server without approving it to run")
	return cmd
}

func parseKeyValues(pairs []string, flag string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	values := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		key, value, found := strings.Cut(pair, "=")
		if !found || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("%s expects KEY=VALUE, got %q", flag, pair)
		}
		values[key] = value
	}
	return values, nil
}

func mcpRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove NAME",
		Short: "Remove an MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, approvals, paths, err := loadMCPState()
			if err != nil {
				return err
			}
			if !cfg.Remove(name) {
				return fmt.Errorf("no MCP server named %q", name)
			}
			if err := cfg.Save(paths.config); err != nil {
				return err
			}
			// Drop the approval too. Leaving it behind would silently
			// re-approve a future server that happened to reuse the name and
			// the command line.
			if approvals.Revoke(name) {
				if err := approvals.Save(paths.approvals); err != nil {
					return err
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", name)
			return nil
		},
	}
}

func mcpEnableDisableCmd(verb string, disabled bool, short string) *cobra.Command {
	return &cobra.Command{
		Use:   verb + " NAME",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, approvals, paths, err := loadMCPState()
			if err != nil {
				return err
			}
			spec, ok := cfg.Get(name)
			if !ok {
				return fmt.Errorf("no MCP server named %q", name)
			}
			spec.Disabled = disabled
			if err := cfg.Save(paths.config); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%sd %s\n", strings.ToUpper(verb[:1])+verb[1:], name)
			if !disabled && !approvals.Allows(spec) {
				fmt.Fprintf(out, "It is not approved to run. Approve with:\n  ollama mcp approve %s\n", name)
			}
			return nil
		},
	}
}

func mcpApproveCmd() *cobra.Command {
	var assumeYes bool

	cmd := &cobra.Command{
		Use:   "approve NAME",
		Short: "Approve an MCP server to run",
		Long: "Approve an MCP server to run.\n\n" +
			"The exact command line is shown before you agree. The approval covers that\n" +
			"command line only: if the server's command, arguments, environment or URL\n" +
			"change afterwards, it must be approved again.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, approvals, paths, err := loadMCPState()
			if err != nil {
				return err
			}
			spec, ok := cfg.Get(name)
			if !ok {
				return fmt.Errorf("no MCP server named %q", name)
			}
			if problem := cfg.Problems()[name]; problem != nil {
				return fmt.Errorf("%s cannot run as configured: %w", name, problem)
			}

			out := cmd.OutOrStdout()
			if approvals.Allows(spec) {
				fmt.Fprintf(out, "%s is already approved to run:\n  %s\n", name, spec.Summary())
				return nil
			}

			// The whole point of this command: show what will actually run,
			// verbatim, before anyone agrees to it.
			fmt.Fprintf(out, "Approving %s will let Ollama run:\n\n  %s\n", name, spec.Summary())
			if len(spec.Env) > 0 {
				fmt.Fprintf(out, "\nwith environment:\n")
				for _, key := range slices.Sorted(maps.Keys(spec.Env)) {
					fmt.Fprintf(out, "  %s=%s\n", key, spec.Env[key])
				}
			}
			if len(spec.Headers) > 0 {
				fmt.Fprintf(out, "\nwith headers:\n")
				for _, key := range slices.Sorted(maps.Keys(spec.Headers)) {
					fmt.Fprintf(out, "  %s: %s\n", key, spec.Headers[key])
				}
			}
			if previous := approvals.Entries[name]; previous.Fingerprint != "" {
				fmt.Fprintf(out, "\nThis server was previously approved to run:\n  %s\n", previous.Summary)
				fmt.Fprintf(out, "It has changed since. Approve only if you made that change.\n")
			}

			if !assumeYes {
				confirmed, err := confirmMCP(cmd.InOrStdin(), out, fmt.Sprintf("\nApprove %s?", name))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintf(out, "Not approved.\n")
					return nil
				}
			}

			approvals.Approve(spec, time.Now())
			if err := approvals.Save(paths.approvals); err != nil {
				return err
			}
			fmt.Fprintf(out, "Approved %s\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "approve without asking")
	return cmd
}

func mcpRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke NAME",
		Short: "Withdraw approval for an MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			_, approvals, paths, err := loadMCPState()
			if err != nil {
				return err
			}
			if !approvals.Revoke(name) {
				return fmt.Errorf("%q was not approved", name)
			}
			if err := approvals.Save(paths.approvals); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Revoked approval for %s. Ollama will no longer run it.\n", name)
			return nil
		},
	}
}

// signInManager builds a manager for the sign-in commands. It is given the
// same approval policy as everything else — a sign-in contacts the server, so
// it goes through the gate that says a server may be contacted at all.
func signInManager() *mcp.Manager {
	approvalsPath, err := mcp.ApprovalsPath()
	if err != nil {
		approvalsPath = ""
	}
	return mcp.NewManager(mcp.Options{
		Approvals: mcp.ApprovalsFile(approvalsPath, nil),
		Tokens:    &mcp.FileTokenStore{},
	})
}

// remoteServer resolves a configured server and refuses one that is not remote,
// so `login` and `logout` fail with a sentence rather than a surprise.
func remoteServer(name string) (*mcp.ServerSpec, error) {
	cfg, _, _, err := loadMCPState()
	if err != nil {
		return nil, err
	}
	spec, ok := cfg.Get(name)
	if !ok {
		return nil, fmt.Errorf("no MCP server named %q", name)
	}
	if problem := cfg.Problems()[name]; problem != nil {
		return nil, fmt.Errorf("%s cannot run as configured: %w", name, problem)
	}
	if spec.URL == "" {
		return nil, fmt.Errorf("%s runs on this machine and has nothing to sign in to", name)
	}
	return spec, nil
}

func mcpLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login NAME",
		Short: "Sign in to a remote MCP server",
		Long: "Sign in to a remote MCP server.\n\n" +
			"A browser is opened for the server's own sign-in page. Ollama never sees\n" +
			"your password: it receives only a token, which it stores and sends back to\n" +
			"that server. This is the only command that opens a browser — servers that\n" +
			"need a sign-in are otherwise reported and left alone.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			spec, err := remoteServer(name)
			if err != nil {
				return err
			}

			manager := signInManager()
			defer manager.Close()

			out := cmd.OutOrStdout()
			// Where the credential ends up is the user's to know before they
			// create one, not after.
			store := &mcp.FileTokenStore{}
			fmt.Fprintf(out, "Signing in to %s (%s).\n", name, spec.URL)
			fmt.Fprintf(out, "Your token will be kept in %s\n", store.Description())
			fmt.Fprintf(out, "Opening your browser; finish there and come back.\n")

			state, err := manager.SignIn(cmd.Context(), spec)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "\nSigned in to %s (%d tools).\n", name, len(state.Tools))
			return nil
		},
	}
}

func mcpLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout NAME",
		Short: "Sign out of a remote MCP server",
		Long: "Sign out of a remote MCP server.\n\n" +
			"The token is revoked at the server and then deleted from this machine. If\n" +
			"the server offers no way to revoke it, the token is still deleted here and\n" +
			"you are told, because withdrawing it then has to be done in that service's\n" +
			"own account settings.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			spec, err := remoteServer(name)
			if err != nil {
				return err
			}

			manager := signInManager()
			defer manager.Close()

			out := cmd.OutOrStdout()
			err = manager.SignOut(cmd.Context(), spec)
			if errors.Is(err, mcp.ErrSignedOutLocallyOnly) {
				fmt.Fprintf(out, "Deleted the stored token for %s.\n", name)
				fmt.Fprintf(out, "It could NOT be revoked at the server, so it may still be valid there:\n  %v\n", err)
				fmt.Fprintf(out, "Withdraw Ollama's access in that service's account settings.\n")
				return nil
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Signed out of %s. The token was revoked at the server and deleted here.\n", name)
			return nil
		},
	}
}

// confirmMCP asks a yes/no question. It defaults to no, and it refuses to
// assume yes when there is nobody there to answer: an approval that can happen
// without a person is not an approval.
func confirmMCP(in io.Reader, out io.Writer, question string) (bool, error) {
	if !isTerminal(in) {
		return false, errors.New("approval needs a terminal to ask on; re-run with --yes if you are sure")
	}

	fmt.Fprintf(out, "%s [y/N] ", question)
	reader := bufio.NewReader(in)
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

// isTerminal reports whether the reader is an interactive terminal. A pipe or a
// test buffer is not.
func isTerminal(in io.Reader) bool {
	file, ok := in.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
