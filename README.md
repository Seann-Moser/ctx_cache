## Usages


## Installing

## Benchmarks
```bash
go install golang.org/x/perf/cmd/benchstat@latest

go test -bench=. cachemonitor_benchmark_test.go | tee cachemonitor.txt

```

```
                                                 │ cachemonitor.txt │          cachemonitor_v2.txt          │
                                                 │      sec/op      │    sec/op     vs base                 │
NewMonitorCacheMonitor/AddGroupKeys-16                11.021µ ± ∞ ¹   4.357µ ± ∞ ¹        ~ (p=1.000 n=1) ²
NewMonitorCacheMonitor/HasGroupKeyBeenUpdated-16       5.226µ ± ∞ ¹   5.972µ ± ∞ ¹        ~ (p=1.000 n=1) ²
NewMonitorCacheMonitor/GetGroupKeys-16               7059.00n ± ∞ ¹   14.83n ± ∞ ¹        ~ (p=1.000 n=1) ²
NewMonitorCacheMonitor/DeleteCache-16                  7.285µ ± ∞ ¹   1.144µ ± ∞ ¹        ~ (p=1.000 n=1) ²
NewMonitorCacheMonitor/UpdateCache-16                 16.943µ ± ∞ ¹   4.101µ ± ∞ ¹        ~ (p=1.000 n=1) ²
geomean                                                8.712µ         1.126µ        -87.07%
```