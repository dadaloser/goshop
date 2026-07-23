# Fault Injection

Run only in an approved non-production namespace with Chaos Mesh installed. Establish a clean SLO baseline, start the k6 workload, then apply one fault at a time with `kubectl apply -f chaos/kubernetes/core-faults.yaml`.

Expected results: an order pod failure causes no more than 1% request errors and recovers within 2 minutes; Elasticsearch latency grows the Outbox backlog without losing MySQL writes and converges after recovery; inventory CPU pressure keeps P95 below 500 ms or triggers the saturation alert. Abort immediately for oversell, payment duplication, data loss, error rate above 5% for 2 minutes, or impact outside the test namespace. Remove faults with `kubectl delete -f chaos/kubernetes/core-faults.yaml` and verify backlog convergence and a synthetic checkout.
