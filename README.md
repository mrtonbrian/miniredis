# miniredis
![Build and Test](https://github.com/mrtonbrian/miniredis/actions/workflows/workflow.yml/badge.svg) 
[![codecov](https://codecov.io/gh/mrtonbrian/miniredis/graph/badge.svg?token=SDMKUHQ5JW)](https://codecov.io/gh/mrtonbrian/miniredis)

A mini version of Redis written in Go. Should hopefully be faster than Redis (due to concurrency and not needing to handle as much).

## Benchmark
### Actual Redis
No disk persistence
```
brianton@brianlenlaptop:~$ redis-benchmark -p 6379 -t set,get -n 10000000 -q -P 512 -c 512
SET: 1704604.62 requests per second
GET: 2375965.00 requests per second
```
### MiniRedis
No disk persistence
```
brianton@brianlenlaptop:~$ redis-benchmark -p 6379 -t set,get -n 10000000 -q -P 512 -c 512
WARN: could not fetch server CONFIG
SET: 1723040.88 requests per second
GET: 5471556.00 requests per second
```
Speedup for `GET` is mostly due to concurrency, I believe (entire table is locked for `SET`, so not much speedup in that case). Should be improved with a sharded concurrent map library?
## TODO list
- [x] Write some basic parser for RESP
- [x] Get an MVP of basic SET / GET functionality
- [x] Run initial `redis-benchmark` on SET / GET
- [x] Implement pipelining
- [x] Run another `redis-benchmark`
- [x] Match / beat redis on the basic SET / GET
- [ ] Implement expiry
- [ ] Implement RDB
- [ ] Implement lists
- [ ] Implement transactions
- [ ] Move to IO_URING
- [ ] Swap map library(?)
- [ ] Implement pub/sub
