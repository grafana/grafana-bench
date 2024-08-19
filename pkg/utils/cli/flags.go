package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)


const usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags 

%s

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// FlagGroup defines a group of flags with a name
type FlagGroup struct {
	name   string
	flags  *pflag.FlagSet
}

// Usage returns the usage of the flags in the FlagGroup
func (g *FlagGroup) Usage() string {
	return fmt.Sprintf("%s:\n%s\n", g.name, g.flags.FlagUsages())
}

// CmdFlags defines the flags of a command
type CmdFlags struct {
	groups []*FlagGroup
}

// AddGroup adds a group of flags to the command's flags
func (c *CmdFlags) AddGroup(name string)  *pflag.FlagSet {
	flags := pflag.NewFlagSet(name, pflag.ContinueOnError)
	c.groups = append(c.groups, &FlagGroup{
		name:  name,
		flags: flags,
	})

	return flags
}

// NewCmdFlags creates a new CmdFlag
func NewCmdFlags() *CmdFlags {
	return &CmdFlags{}
}

// SetCmd binds the CmdFlags to a command, setting the Usage template to
// list flags by group
func (c *CmdFlags)SetCmd(cmd *cobra.Command) {
	
	cmdFlags := pflag.NewFlagSet("other", pflag.ContinueOnError)
	hasFlags := false
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		cmdFlags.AddFlag(f)
		hasFlags = true
	})
	if hasFlags {
		c.groups = append(c.groups, &FlagGroup{name: "General", flags: cmdFlags})
	}

	groups := c.groups
	slices.SortFunc(groups, func(a, b *FlagGroup) int {
		return strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
	})

	for _, g := range groups {
		g.flags.VisitAll(func(f *pflag.Flag) {
			cmd.Flags().AddFlag(f)
		})
	}

	cmd.SetUsageTemplate(c.usageTemplate())
}

// usageTemplate concatenates the usage for each group and embeds them into the command's help
func (c *CmdFlags) usageTemplate() string {
	var usage string

	for _, g := range c.groups {
		usage += g.Usage()
	}

	return fmt.Sprintf(usageTemplate, usage)
}

