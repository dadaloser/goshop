package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	cliflag "goshop/pkg/common/cli/flag"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type testOptions struct {
	Value string `mapstructure:"value"`
}

func (*testOptions) Flags() cliflag.NamedFlagSets { return cliflag.NamedFlagSets{} }

func (*testOptions) Validate() []error { return nil }

func TestAppConfigIsolatedPerInstance(t *testing.T) {
	firstConfig := writeTestConfig(t, "value: first\n")
	secondConfig := writeTestConfig(t, "value: second\n")

	firstOptions := &testOptions{}
	firstRan := false
	first := NewApp(
		"first",
		"first-server",
		WithOptions(firstOptions),
		WithViper(viper.New()),
		WithFlagSet(pflag.NewFlagSet("first", pflag.ContinueOnError)),
		WithSilence(),
		WithRunFunc(func(context.Context, string) error {
			firstRan = true
			return nil
		}),
	)
	first.Command().SetArgs([]string{"--config", firstConfig})
	if err := first.Run(); err != nil {
		t.Fatalf("first Run() error = %v, want nil", err)
	}

	secondOptions := &testOptions{}
	secondRan := false
	second := NewApp(
		"second",
		"second-server",
		WithOptions(secondOptions),
		WithViper(viper.New()),
		WithFlagSet(pflag.NewFlagSet("second", pflag.ContinueOnError)),
		WithSilence(),
		WithRunFunc(func(context.Context, string) error {
			secondRan = true
			return nil
		}),
	)
	second.Command().SetArgs([]string{"--config", secondConfig})
	if err := second.Run(); err != nil {
		t.Fatalf("second Run() error = %v, want nil", err)
	}

	if !firstRan || !secondRan {
		t.Fatalf("run callbacks = first:%t second:%t, want both true", firstRan, secondRan)
	}
	if firstOptions.Value != "first" {
		t.Fatalf("first config value = %q, want first", firstOptions.Value)
	}
	if secondOptions.Value != "second" {
		t.Fatalf("second config value = %q, want second", secondOptions.Value)
	}
}

func TestAppReturnsConfigLoadError(t *testing.T) {
	ran := false
	a := NewApp(
		"missing",
		"missing-server",
		WithOptions(&testOptions{}),
		WithViper(viper.New()),
		WithFlagSet(pflag.NewFlagSet("missing", pflag.ContinueOnError)),
		WithSilence(),
		WithRunFunc(func(context.Context, string) error {
			ran = true
			return nil
		}),
	)
	a.Command().SetArgs([]string{"--config", filepath.Join(t.TempDir(), "missing.yaml")})

	if err := a.Run(); err == nil {
		t.Fatal("Run() error = nil, want configuration load error")
	}
	if ran {
		t.Fatal("run callback was called after configuration load failure")
	}
}

func TestCommandReturnsRunError(t *testing.T) {
	want := errors.New("command failed")
	command := NewCommand("fail", "fail", WithCommandRunFunc(func([]string) error {
		return want
	}))
	if err := command.cobraCommand().Execute(); !errors.Is(err, want) {
		t.Fatalf("Execute() error = %v, want command error", err)
	}
}

func writeTestConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
