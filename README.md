# dns-checker

The tool for DNS diagnostics.

#### Usage
```
$ dns-checker --help
Usage of dns-checker:
  -c, --count int             number of requests to send. Default is to run until ^C
  -i, --interval duration     interval between requests (default 100ms)
  -n, --name string           resource record name
  -s, --nameservers strings   comma separated nameservers names with port (default [10.0.0.1:53])
  -r, --no-recursion          disable recursion desired flag
  -e, --only-errors           show only errors
  -q, --query-type string     query type to use (A, AAAA, SOA, CNAME...) (default "A")
  -t, --timeout duration      query timeout (default 2s)
```

#### Example

Query DNS record mixanemca.ru with type A for two DNS nameservers (1.1.1.1 and 8.8.8.8) two times.

```
$ dns-checker --name mixanemca.ru. --query-type A --nameservers "1.1.1.1:53,8.8.8.8:53" --count 2
2021-07-15 19:43:38.806987 +0300 MSK m=+0.174620341
Nameserver: 8.8.8.8:53
mixanemca.ru.	3599	IN	A	134.209.90.80

2021-07-15 19:43:39.415163 +0300 MSK m=+0.782800789
Nameserver: 1.1.1.1:53
mixanemca.ru.	3600	IN	A	134.209.90.80

2021-07-15 19:43:39.419918 +0300 MSK m=+0.787556393
Nameserver: 1.1.1.1:53
mixanemca.ru.	3597	IN	A	134.209.90.80

2021-07-15 19:43:40.837345 +0300 MSK m=+2.204993198
Nameserver: 8.8.8.8:53
exchange failed: read udp 192.168.44.99:51871->8.8.8.8:53: i/o timeout (rtt: 2.000037788s)

--- mixanemca.ru. dns check statistics ---
1 error (25.00%) and 3 success (75.00%) for 2 nameservers.
round-trip min/avg/max = 4.638/251.414/678.894 ms
```
