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


```go
                                                 │  go-json.txt  │               json.txt                │
                                                 │    sec/op     │    sec/op      vs base                │
NewMonitorCacheMonitor/AddGroupKeys-16              10.22µ ± ∞ ¹    12.45µ ± ∞ ¹       ~ (p=1.000 n=1) ²
NewMonitorCacheMonitor/HasGroupKeyBeenUpdated-16    4.280µ ± ∞ ¹    4.072µ ± ∞ ¹       ~ (p=1.000 n=1) ²
NewMonitorCacheMonitor/GetGroupKeys-16              6.379µ ± ∞ ¹    7.822µ ± ∞ ¹       ~ (p=1.000 n=1) ²
NewMonitorCacheMonitor/DeleteCache-16               6.476µ ± ∞ ¹    9.690µ ± ∞ ¹       ~ (p=1.000 n=1) ²
NewMonitorCacheMonitor/UpdateCache-16               13.01µ ± ∞ ¹    17.95µ ± ∞ ¹       ~ (p=1.000 n=1) ²
NewMonitorV2Monitor/AddGroupKeys-16                 3.361µ ± ∞ ¹    3.518µ ± ∞ ¹       ~ (p=1.000 n=1) ²
NewMonitorV2Monitor/HasGroupKeyBeenUpdated-16       3.579µ ± ∞ ¹    3.360µ ± ∞ ¹       ~ (p=1.000 n=1) ²
NewMonitorV2Monitor/GetGroupKeys-16                 9.838n ± ∞ ¹   10.020n ± ∞ ¹       ~ (p=1.000 n=1) ²
NewMonitorV2Monitor/DeleteCache-16                  3.255µ ± ∞ ¹    3.424µ ± ∞ ¹       ~ (p=1.000 n=1) ²
NewMonitorV2Monitor/UpdateCache-16                  3.566µ ± ∞ ¹    3.469µ ± ∞ ¹       ~ (p=1.000 n=1) ²
CheckPrimaryType/int-16                            10.050n ± ∞ ¹    9.503n ± ∞ ¹       ~ (p=1.000 n=1) ²
CheckPrimaryType/float64-16                         11.02n ± ∞ ¹    10.23n ± ∞ ¹       ~ (p=1.000 n=1) ²
CheckPrimaryType/string-16                          10.96n ± ∞ ¹    10.50n ± ∞ ¹       ~ (p=1.000 n=1) ²
CheckPrimaryType/bool-16                            9.525n ± ∞ ¹    9.588n ± ∞ ¹       ~ (p=1.000 n=1) ²
CheckPrimaryType/slice-16                           11.09n ± ∞ ¹    10.63n ± ∞ ¹       ~ (p=1.000 n=1) ²
CheckPrimaryType/struct-16                          12.74n ± ∞ ¹    10.11n ± ∞ ¹       ~ (p=1.000 n=1) ²
ConvertBytesToType/int-16                           66.59n ± ∞ ¹    75.10n ± ∞ ¹       ~ (p=1.000 n=1) ²
ConvertBytesToType/float64-16                       89.16n ± ∞ ¹    85.09n ± ∞ ¹       ~ (p=1.000 n=1) ²
ConvertBytesToType/bool-16                          37.65n ± ∞ ¹    35.71n ± ∞ ¹       ~ (p=1.000 n=1) ²
ConvertBytesToType/string-16                        121.5n ± ∞ ¹    111.6n ± ∞ ¹       ~ (p=1.000 n=1) ²
GetTypeReflect/int-16                               8.340n ± ∞ ¹    8.538n ± ∞ ¹       ~ (p=1.000 n=1) ²
GetTypeReflect/float64-16                           8.000n ± ∞ ¹    8.556n ± ∞ ¹       ~ (p=1.000 n=1) ²
GetTypeReflect/string-16                            8.328n ± ∞ ¹    8.043n ± ∞ ¹       ~ (p=1.000 n=1) ²
GetTypeReflect/bool-16                              8.213n ± ∞ ¹    8.149n ± ∞ ¹       ~ (p=1.000 n=1) ²
GetTypeSprintf/int-16                               93.39n ± ∞ ¹    90.04n ± ∞ ¹       ~ (p=1.000 n=1) ²
GetTypeSprintf/float64-16                           141.6n ± ∞ ¹    146.4n ± ∞ ¹       ~ (p=1.000 n=1) ²
GetTypeSprintf/string-16                            181.8n ± ∞ ¹    192.2n ± ∞ ¹       ~ (p=1.000 n=1) ²
GetTypeSprintf/bool-16                              89.86n ± ∞ ¹    96.69n ± ∞ ¹       ~ (p=1.000 n=1) ²
geomean                                             140.7n          144.6n        +2.78%

```