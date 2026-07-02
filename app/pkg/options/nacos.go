package options

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"
)

type NacosOptions struct {
	Host      string `mapstructure:"host" json:"host"`
	Port      uint64 `mapstructure:"port" json:"port"`
	Namespace string `mapstructure:"namespace" json:"namespace"`
	User      string `mapstructure:"user" json:"user"`
	Password  string `mapstructure:"password" json:"password"`
	DataId    string `mapstructure:"dataid" json:"dataid"`
	Group     string `mapstructure:"group" and:"group"`
}

// 默认配置
func NewNacosOptions() *NacosOptions {
	return &NacosOptions{
		Host:      "127.0.0.1",
		Port:      8848,
		Namespace: "public",
		User:      "nacos",
		Password:  "nacos",
		DataId:    "flow",
		Group:     "sentinel-go",
	}
}

func (n *NacosOptions) Validate() []error {
	var errs []error
	if n.Host == "" {
		errs = append(errs, errors.New("nacos.host is required"))
	}
	if n.Port == 0 || n.Port > 65535 {
		errs = append(errs, fmt.Errorf("nacos.port must be between 1 and 65535, got %d", n.Port))
	}
	if n.Namespace == "" {
		errs = append(errs, errors.New("nacos.namespace is required"))
	}
	if n.DataId == "" {
		errs = append(errs, errors.New("nacos.dataid is required"))
	}
	if n.Group == "" {
		errs = append(errs, errors.New("nacos.group is required"))
	}

	return errs
}

func (n *NacosOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&n.Host, "nacos.host", n.Host, "nacos host")
	fs.Uint64Var(&n.Port, "nacos.port", n.Port, "nacos port")
	fs.StringVar(&n.Namespace, "nacos.namespace", n.Namespace, "nacos namespace")
	fs.StringVar(&n.User, "nacos.user", n.User, "nacos user")
	fs.StringVar(&n.Password, "nacos.password", n.Password, "nacos password")
	fs.StringVar(&n.DataId, "nacos.dataid", n.DataId, "nacos dataid")
	fs.StringVar(&n.Group, "nacos.group", n.Group, "nacos group")
}
