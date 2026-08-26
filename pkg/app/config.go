package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"goshop/pkg/common/util/homedir"

	"github.com/gosuri/uitable"
	"github.com/spf13/pflag"
)

const configFlagName = "config"

// addConfigFlag adds the per-App configuration flag to fs. It deliberately
// never touches pflag.CommandLine.
func (a *App) addConfigFlag(fs *pflag.FlagSet) {
	if fs.Lookup(configFlagName) == nil {
		fs.StringVarP(&a.configFile, configFlagName, "c", "", "Read configuration from specified `FILE`, "+
			"support JSON, TOML, YAML, HCL, or Java properties formats.")
	}
}

func (a *App) loadConfig() error {
	a.viper.AutomaticEnv()
	a.viper.SetEnvPrefix(strings.ReplaceAll(strings.ToUpper(a.basename), "-", "_"))
	a.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	configFile := a.configFile
	if flag := a.appFlags.Lookup(configFlagName); flag != nil {
		configFile = flag.Value.String()
	}
	if configFile != "" {
		a.viper.SetConfigFile(configFile)
	} else {
		a.viper.AddConfigPath(".")
		if names := strings.Split(a.basename, "-"); len(names) > 1 {
			a.viper.AddConfigPath(filepath.Join(homedir.HomeDir(), "."+names[0]))
		}
		a.viper.SetConfigName(a.basename)
	}

	if err := a.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read configuration file %q: %w", configFile, err)
	}
	return nil
}

func (a *App) printConfig() {
	keys := a.viper.AllKeys()
	if len(keys) > 0 {
		_, _ = fmt.Fprintf(a.cmd.OutOrStdout(), "%v Configuration items:\n", progressMessage)
		table := uitable.New()
		table.Separator = " "
		table.MaxColWidth = 80
		table.RightAlign(0)
		for _, k := range keys {
			value := a.viper.Get(k)
			if isSensitiveKey(k) {
				value = redactedValue
			}
			table.AddRow(fmt.Sprintf("%s:", k), value)
		}
		_, _ = fmt.Fprint(a.cmd.OutOrStdout(), table)
	}
}
