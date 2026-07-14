package options

import "goshop/gmicro/server/rpcserver"

// RPCSecurityOptions remains as a compatibility alias while the security
// policy implementation lives in gmicro.
type RPCSecurityOptions = rpcserver.SecurityPolicy

func NewRPCSecurityOptions() *RPCSecurityOptions {
	return rpcserver.NewSecurityPolicy()
}
