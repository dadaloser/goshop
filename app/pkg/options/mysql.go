package options

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/pflag"
)

type MySQLOptions struct {
	Host                  string        `mapstructure:"host" json:"host,omitempty"`
	Port                  string        `mapstructure:"port" json:"port,omitempty"`
	Username              string        `mapstructure:"username" json:"username,omitempty"`
	Password              string        `mapstructure:"password" json:"password,omitempty"`
	Database              string        `mapstructure:"database" json:"database"`
	MaxIdleConnections    int           `mapstructure:"max-idle-connections" json:"max-idle-connections,omitempty"`
	MaxOpenConnections    int           `mapstructure:"max-open-connections" json:"max-open-connections,omitempty"`
	MaxConnectionLifetime time.Duration `mapstructure:"max-connection-life-time" json:"max-connection-life-time,omitempty"`
	LogLevel              int           `mapstructure:"log-level" json:"log-level"`
}

// NewMySQLOptions create a `zero` value instance.
func NewMySQLOptions() *MySQLOptions {
	return &MySQLOptions{
		Host:                  "127.0.0.1",
		Port:                  "3306",
		Username:              "",
		Password:              "",
		Database:              "",
		MaxIdleConnections:    100,
		MaxOpenConnections:    100,
		MaxConnectionLifetime: time.Duration(10) * time.Second,
		LogLevel:              1, // Silent
	}
}

func (o *MySQLOptions) Validate() []error {
	var errs []error
	if o.Host == "" {
		errs = append(errs, errors.New("mysql.host is required"))
	}
	port, err := strconv.Atoi(o.Port)
	if err != nil {
		errs = append(errs, fmt.Errorf("mysql.port must be numeric: %w", err))
	} else if port <= 0 || port > 65535 {
		errs = append(errs, fmt.Errorf("mysql.port must be between 1 and 65535, got %d", port))
	}
	if o.Username == "" {
		errs = append(errs, errors.New("mysql.username is required"))
	}
	if o.Password == "" {
		errs = append(errs, errors.New("mysql.password is required"))
	}
	if o.Database == "" {
		errs = append(errs, errors.New("mysql.database is required"))
	}
	if o.MaxIdleConnections < 0 {
		errs = append(errs, errors.New("mysql.max-idle-connections must not be negative"))
	}
	if o.MaxOpenConnections <= 0 {
		errs = append(errs, errors.New("mysql.max-open-connections must be positive"))
	}
	if o.MaxIdleConnections > o.MaxOpenConnections {
		errs = append(errs, errors.New("mysql.max-idle-connections must not exceed mysql.max-open-connections"))
	}
	if o.MaxConnectionLifetime <= 0 {
		errs = append(errs, errors.New("mysql.max-connection-life-time must be positive"))
	}

	return errs
}

func (o *MySQLOptions) ValidateStartup() error {
	if o.Host == "" {
		return errors.New("mysql.host is required")
	}
	port, err := strconv.Atoi(o.Port)
	if err != nil {
		return fmt.Errorf("mysql.port must be numeric: %w", err)
	}
	if port <= 0 || port > 65535 {
		return fmt.Errorf("mysql.port must be between 1 and 65535, got %d", port)
	}
	if o.Username == "" {
		return errors.New("mysql.username is required")
	}
	if o.Password == "" {
		return errors.New("mysql.password is required")
	}
	if o.Database == "" {
		return errors.New("mysql.database is required")
	}
	if o.MaxIdleConnections < 0 {
		return errors.New("mysql.max-idle-connections must not be negative")
	}
	if o.MaxOpenConnections <= 0 {
		return errors.New("mysql.max-open-connections must be positive")
	}
	if o.MaxIdleConnections > o.MaxOpenConnections {
		return errors.New("mysql.max-idle-connections must not exceed mysql.max-open-connections")
	}
	if o.MaxConnectionLifetime <= 0 {
		return errors.New("mysql.max-connection-life-time must be positive")
	}

	return nil
}

// AddFlags adds flags related to mysql storage for a specific APIServer to the specified FlagSet.
func (mo *MySQLOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&mo.Host, "mysql.host", mo.Host, ""+
		"MySQL service host address. If left blank, the following related mysql options will be ignored.")

	fs.StringVar(&mo.Port, "mysql.port", mo.Port, ""+
		"MySQL service port")

	fs.StringVar(&mo.Username, "mysql.username", mo.Username, ""+
		"Username for access to mysql service.")

	fs.StringVar(&mo.Password, "mysql.password", mo.Password, ""+
		"Password for access to mysql, should be used pair with password.")

	fs.StringVar(&mo.Database, "mysql.database", mo.Database, ""+
		"Database name for the server to use.")

	fs.IntVar(&mo.MaxIdleConnections, "mysql.max-idle-connections", mo.MaxOpenConnections, ""+
		"Maximum idle connections allowed to connect to mysql.")

	fs.IntVar(&mo.MaxOpenConnections, "mysql.max-open-connections", mo.MaxOpenConnections, ""+
		"Maximum open connections allowed to connect to mysql.")

	fs.DurationVar(&mo.MaxConnectionLifetime, "mysql.max-connection-life-time", mo.MaxConnectionLifetime, ""+
		"Maximum connection life time allowed to connecto to mysql.")

	fs.IntVar(&mo.LogLevel, "mysql.log-mode", mo.LogLevel, ""+
		"Specify gorm log level.")
}
