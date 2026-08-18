package clientinterceptors

import "testing"

func TestRPCClientDurationBucketsCoverDefaultResilienceTimeout(t *testing.T) {
	if len(rpcClientDurationBuckets) == 0 {
		t.Fatal("rpcClientDurationBuckets is empty")
	}
	if got := rpcClientDurationBuckets[len(rpcClientDurationBuckets)-1]; got < 5000 {
		t.Errorf("largest RPC client duration bucket = %vms, want at least 5000ms", got)
	}
	foundAboveDefaultTimeout := false
	for _, bucket := range rpcClientDurationBuckets {
		if bucket > 2000 {
			foundAboveDefaultTimeout = true
			break
		}
	}
	if !foundAboveDefaultTimeout {
		t.Fatal("RPC client duration buckets have no finite bucket above the default 2000ms timeout")
	}
}
