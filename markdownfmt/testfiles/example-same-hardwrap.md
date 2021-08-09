The `thanos rule` command evaluates Prometheus recording and alerting rules
against chosen query API via repeated `--query` (or FileSD via `--query.sd`).
If more than one query is passed, round robin balancing is performed.

Rule results are written back to disk in the Prometheus 2.0 storage format.
Rule nodes at the same time participate in the system as source store nodes,
which means that they expose StoreAPI and *upload* their generated TSDB blocks
to an object store.

You can think of Rule as a simplified Prometheus that does not require
a sidecar and does not scrape and do PromQL evaluation (no QueryAPI).
