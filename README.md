# miniredis
![Build and Test](https://github.com/mrtonbrian/miniredis/actions/workflows/workflow.yml/badge.svg) 
[![codecov](https://codecov.io/gh/mrtonbrian/miniredis/graph/badge.svg?token=SDMKUHQ5JW)](https://codecov.io/gh/mrtonbrian/miniredis)
[![Go Report Card](https://goreportcard.com/badge/github.com/mrtonbrian/miniredis)](https://goreportcard.com/report/github.com/mrtonbrian/miniredis)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)

A mini version of Redis written in Go. Slightly faster than Redis in some situations. Not meant for production use.

## Benchmark
Here is a mini benchmark on basic `SET`/`GET` commands that I ran on my laptop.
### Actual Redis
No disk persistence
```bash
redis-server --port 6379 --appendonly no
```
Result:
```
brianton@brianlenlaptop:~$ redis-benchmark -p 6379 -t set,get -n 10000000 -q -P 512 -c 512
SET: 1704604.62 requests per second
GET: 2375965.00 requests per second
```
### MiniRedis
No disk persistence
```
./miniredis.sh
```
Result:
```
brianton@brianlenlaptop:~$ redis-benchmark -p 6379 -t set,get -n 10000000 -q -P 512 -c 512
WARN: could not fetch server CONFIG
SET: 1723040.88 requests per second
GET: 5471556.00 requests per second
```
Speedup for `GET` is mostly due to concurrency, I believe (entire table is locked for `SET`, so not much speedup in that case). Could be improved with a sharded concurrent map library? My initial trials say no, but intuitively, it should work.

Also do take these benchmarks with a grain of salt - these were done on the Lenovo Yoga Slim 7i Aura edition (Intel Ultra 7 258v, 32GB), but definitely not a sanitized environment (many processes open in the background). Here, I believe we're benefitting a lot from the 8 cores and high memory speed (due to the memory being on the CPU package itself). Thus, your mileage may vary, and definitely do **not** use this for production.

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
