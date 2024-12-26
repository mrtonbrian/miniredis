# miniredis
A mini version of Redis written in Go. Should hopefully be faster than Redis (due to concurrency and not needing to handle as much).

## Benchmark
### Actual Redis
```
brianton@brian-mini:~$ redis-benchmark -p 6379 -t set,get -n 10000000 -q -P 512 -c 512
SET: 1334986.50 requests per second, p50=163.967 msec
GET: 1707715.88 requests per second, p50=139.263 msec
```
### MiniRedis

## TODO list
- [ ] Write some basic parser for RESP
- [ ] Get an MVP of basic SET / GET functionality
- [ ] Run initial `redis-benchmark` on SET / GET
- [ ] Implement expiry
- [ ] Implement RDB
- [ ] Implement lists
- [ ] Implement pipelining
- [ ] Run another `redis-benchmark` (with / without pipelining)
- [ ] Implement transactions
- [ ] Move to IO_URING
- [ ] Swap map library(?)
- [ ] Implement pub/sub
