package app

import (
	"context"
	"fmt"
	cliflag "goshop/pkg/common/cli/flag"
	"goshop/pkg/common/cli/globalflag"
	"goshop/pkg/common/term"
	"goshop/pkg/common/version"
	"goshop/pkg/errors"
	"os"

	//controller(参数校验) ->service(具体的业务逻辑) -> data(数据库的接口)
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"goshop/pkg/log"
)

var (
	progressMessage = color.GreenString("==>")
	//nolint:unused
	usageTemplate = fmt.Sprintf(`%s{{if .Runnable}}
  %s{{end}}{{if .HasAvailableSubCommands}}
  %s{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  %s {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

%s{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "%s --help" for more information about a command.{{end}}
`,
		color.CyanString("Usage:"),
		color.GreenString("{{.UseLine}}"),
		color.GreenString("{{.CommandPath}} [command]"),
		color.CyanString("Aliases:"),
		color.CyanString("Examples:"),
		color.CyanString("Available Commands:"),
		color.GreenString("{{rpad .Name .NamePadding }}"),
		color.CyanString("Flags:"),
		color.CyanString("Global Flags:"),
		color.CyanString("Additional help topics:"),
		color.GreenString("{{.CommandPath}} [command]"),
	)
)

// App is the main structure of a cli application.
// It is recommended that an app be created with the app.NewApp() function.
type App struct {
	basename    string
	name        string
	description string
	options     CliOptions
	runFunc     RunFunc
	silence     bool
	noVersion   bool
	noConfig    bool
	commands    []*Command
	args        cobra.PositionalArgs
	cmd         *cobra.Command
	viper       *viper.Viper
	appFlags    *pflag.FlagSet
	configFile  string
	versionFlag string
}

// Option defines optional parameters for initializing the application
// structure.
type Option func(*App)

// WithViper injects the configuration instance used by App. Each App should
// receive its own viper.New() instance when callers need explicit isolation.
func WithViper(v *viper.Viper) Option {
	return func(a *App) {
		if v != nil {
			a.viper = v
		}
	}
}

// WithFlagSet injects App-owned flags. The set must not be shared by multiple
// App instances because Cobra and pflag mutate it while executing commands.
func WithFlagSet(fs *pflag.FlagSet) Option {
	return func(a *App) {
		if fs != nil {
			a.appFlags = fs
		}
	}
}

// WithOptions to open the application's function to read from the command line
// or read parameters from the configuration file.
func WithOptions(opt CliOptions) Option {
	return func(a *App) {
		a.options = opt
	}
}

// RunFunc defines the application's startup callback function.
type RunFunc func(ctx context.Context, basename string) error

// WithRunFunc is used to set the application startup callback function option.
func WithRunFunc(run RunFunc) Option {
	return func(a *App) {
		a.runFunc = run
	}
}

// WithDescription is used to set the description of the application.
func WithDescription(desc string) Option {
	return func(a *App) {
		a.description = desc
	}
}

// WithSilence sets the application to silent mode, in which the program startup
// information, configuration information, and version information are not
// printed in the console.
func WithSilence() Option {
	return func(a *App) {
		a.silence = true
	}
}

// WithNoVersion set the application does not provide version flag.
func WithNoVersion() Option {
	return func(a *App) {
		a.noVersion = true
	}
}

// WithNoConfig set the application does not provide config flag.
func WithNoConfig() Option {
	return func(a *App) {
		a.noConfig = true
	}
}

// WithValidArgs set the validation function to valid non-flag arguments.
func WithValidArgs(args cobra.PositionalArgs) Option {
	return func(a *App) {
		a.args = args
	}
}

// WithDefaultValidArgs set default validation function to valid non-flag arguments.
func WithDefaultValidArgs() Option {
	return func(a *App) {
		a.args = func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}

			return nil
		}
	}
}

// NewApp creates a new application instance based on the given application name,
// binary name, and other options.
func NewApp(name string, basename string, opts ...Option) *App {
	a := &App{
		name:     name,
		basename: basename,
		viper:    viper.New(),
		appFlags: pflag.NewFlagSet(basename, pflag.ContinueOnError),
	}

	for _, o := range opts {
		o(a)
	}

	a.buildCommand()

	return a
}

func (a *App) buildCommand() {
	cmd := cobra.Command{
		Use:   FormatBaseName(a.basename),
		Short: a.name,
		Long:  a.description,
		// stop printing usage when the command errors
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          a.args,
	}
	cmd.PersistentPreRunE = a.initializeCommand
	// cmd.SetUsageTemplate(usageTemplate)
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.Flags().SortFlags = true
	cliflag.InitFlags(cmd.Flags())

	if len(a.commands) > 0 {
		for _, command := range a.commands {
			cmd.AddCommand(command.cobraCommand())
		}
		cmd.SetHelpCommand(helpCommand(a.name))
	}
	if a.runFunc != nil {
		cmd.RunE = a.runCommand
	}

	var namedFlagSets cliflag.NamedFlagSets
	if a.options != nil {
		namedFlagSets = a.options.Flags()
		usageFmt := "Usage:\n  %s\n"
		cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
		cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
			cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
		})
		cmd.SetUsageFunc(func(cmd *cobra.Command) error {
			_, _ = fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
			cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)

			return nil
		})
	}

	globalFlags := namedFlagSets.FlagSet("global")
	if !a.noVersion {
		a.addVersionFlag(globalFlags)
	}

	if !a.noConfig {
		a.addConfigFlag(a.appFlags)
	}
	globalFlags.AddFlagSet(a.appFlags)

	globalflag.AddGlobalFlags(globalFlags, cmd.Name())
	for _, flags := range namedFlagSets.FlagSets {
		cmd.Flags().AddFlagSet(flags)
	}

	a.cmd = &cmd
}

// Run executes the command and returns configuration or application errors to
// its caller. Process exit belongs to the composition root in cmd/.
func (a *App) Run() error {
	return a.cmd.Execute()
}

// Command returns cobra command instance inside the application.
func (a *App) Command() *cobra.Command {
	return a.cmd
}

func (a *App) runCommand(cmd *cobra.Command, args []string) error {
	printWorkingDir()
	cliflag.PrintFlags(cmd.Flags())
	if !a.noVersion {
		if a.versionRequested() {
			return a.printVersion(cmd)
		}
	}

	if !a.noConfig {
		if err := a.applyOptionRules(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	if !a.silence {
		log.Infof("%v Starting %s ...", progressMessage, a.name)
		if !a.noVersion {
			log.Infof("%v Version: `%s`", progressMessage, version.Get().ToJSON())
		}
		if !a.noConfig {
			log.Infof("%v Config file used: `%s`", progressMessage, a.viper.ConfigFileUsed())
		}
	}
	// run application
	if a.runFunc != nil {
		return a.runFunc(cmd.Context(), a.basename)
	}

	return nil
}

func (a *App) initializeCommand(cmd *cobra.Command, _ []string) error {
	if a.noConfig || a.versionRequested() {
		return nil
	}
	if a.options == nil {
		return fmt.Errorf("app options are required when configuration is enabled")
	}
	if err := a.viper.BindPFlags(cmd.Flags()); err != nil {
		return fmt.Errorf("bind command flags: %w", err)
	}
	if err := a.loadConfig(); err != nil {
		return err
	}
	if err := a.viper.Unmarshal(a.options); err != nil {
		return fmt.Errorf("decode configuration: %w", err)
	}
	if !a.silence {
		a.printConfig()
	}
	return nil
}

func (a *App) addVersionFlag(fs *pflag.FlagSet) {
	if fs.Lookup("version") != nil {
		return
	}
	fs.StringVar(&a.versionFlag, "version", "", "Print version information and quit.")
	fs.Lookup("version").NoOptDefVal = "true"
}

func (a *App) versionRequested() bool {
	return a.versionFlag == "true" || a.versionFlag == "raw"
}

func (a *App) printVersion(cmd *cobra.Command) error {
	if a.versionFlag == "raw" {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%#v\n", version.Get())
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", version.Get())
	return err
}

func (a *App) applyOptionRules() error {
	if completableOptions, ok := a.options.(CompletableOptions); ok {
		if err := completableOptions.Complete(); err != nil {
			return err
		}
	}

	if errs := a.options.Validate(); len(errs) != 0 {
		return errors.NewAggregate(errs)
	}

	if startupOptions, ok := a.options.(StartupValidatableOptions); ok {
		if err := startupOptions.ValidateStartup(); err != nil {
			return err
		}
	}

	if printableOptions, ok := a.options.(PrintableOptions); ok && !a.silence {
		configText := printableOptions.String()
		if secureOptions, ok := a.options.(SecurePrintableOptions); ok {
			configText = secureOptions.SafeString()
		} else {
			configText = RedactJSON(configText)
		}
		log.Infof("%v Config: `%s`", progressMessage, configText)
	}

	return nil
}

func printWorkingDir() {
	wd, _ := os.Getwd()
	log.Infof("%v WorkingDir: %s", progressMessage, wd)
}
