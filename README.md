# miniredis
A mini version of Redis written in Go. Should hopefully be faster than Redis (due to concurrency and not needing to handle as much).

## Benchmark
### Actual Redis
No pipelining, no disk persistence
```
brianton@brian-mini:~$ redis-benchmark -p 6379 -t set,get -n 1000000 -q -c 512
SET: 44616.96 requests per second, p50=5.463 msec
GET: 44748.73 requests per second, p50=5.383 msec
```
### MiniRedis
No pipelining, no disk persistence
```
brianton@brian-mini:~$ redis-benchmark -p 6379 -t set,get -n 1000000 -q -c 512
WARNING: Could not fetch server CONFIG
SET: 36330.61 requests per second, p50=7.063 msec
GET: 36670.33 requests per second, p50=6.991 msec
```
Will improve this soon!
## TODO list
- [x] Write some basic parser for RESP
- [x] Get an MVP of basic SET / GET functionality
- [x] Run initial `redis-benchmark` on SET / GET
- [x] Implement pipelining
- [ ] Run another `redis-benchmark` (with / without pipelining)
- [ ] Match / beat redis on the basic SET / GET with pipelining
    - Optional: Match / beat redis on the basic SET / GET without pipelining
- [ ] Implement expiry
- [ ] Implement RDB
- [ ] Implement lists
- [ ] Implement transactions
- [ ] Move to IO_URING
- [ ] Swap map library(?)
- [ ] Implement pub/sub
