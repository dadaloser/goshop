package client

import (
	"fmt"
	"strings"
)

const (
	ServiceGoods     = "goshop-goods-srv"
	ServiceInventory = "goshop-inventory-srv"
	ServiceOrder     = "goshop-order-srv"
	ServiceUser      = "goshop-user-srv"
	ServiceReview    = ServiceOrder
)

func ServiceEndpoint(service string) string {
	service = strings.TrimSpace(service)
	if service == "" {
		panic("service name is empty")
	}
	if strings.Contains(service, "://") {
		return service
	}
	return fmt.Sprintf("discovery:///%s", service)
}
