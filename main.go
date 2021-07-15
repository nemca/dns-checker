/*
Copyright © 2021 Michael Bruskov <mixanemca@yandex.ru>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"fortio.org/fortio/stats"
	"github.com/miekg/dns"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
)

// flags
var (
	name        string
	nameservers []string
	queryType   string
	noRecursion bool
	onlyError   bool
	timeout     time.Duration
	interval    time.Duration
	count       int
	// version string
)

// stats
var (
	errorCount   int
	successCount int
)

func init() {
	pflag.StringVarP(&name, "name", "n", "", "resource record name")
	pflag.StringSliceVarP(&nameservers, "nameservers", "s", []string{"10.0.0.1:53"}, "comma separated nameservers names with port")
	pflag.StringVarP(&queryType, "query-type", "q", "A", "query type to use (A, AAAA, SOA, CNAME...)")
	pflag.BoolVarP(&noRecursion, "no-recursion", "r", false, "disable recursion desired flag")
	pflag.BoolVarP(&onlyError, "only-errors", "e", false, "show only errors")
	pflag.DurationVarP(&timeout, "timeout", "t", 2*time.Second, "query timeout")
	pflag.DurationVarP(&interval, "interval", "i", 100*time.Millisecond, "interval between requests")
	pflag.IntVarP(&count, "count", "c", 0, "number of requests to send. Default is to run until ^C")
}

func main() {
	pflag.Parse()

	if len(name) == 0 || len(nameservers) < 1 {
		pflag.Usage()
		os.Exit(1)
	}

	qt, exists := dns.StringToType[strings.ToUpper(queryType)]
	if !exists {
		fmt.Printf("Invalid query type %q\n", queryType)
		os.Exit(1)
	}

	// Create DNS message
	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.SetQuestion(name, qt)
	msg.RecursionDesired = !noRecursion

	// Channel for terminating by signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	// Channel for stop goroutins
	done := make(chan int)

	// Stats
	statHistogram := stats.NewHistogram(0, 0.1)

	go func(done chan int) {
		<-interrupt
		close(done)
	}(done)

	g := new(errgroup.Group)

	for _, nserver := range nameservers {
		ns := nserver // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			pinger(ns, msg, interval, statHistogram, done)
			return nil
		})
	}
	// Wait errgroup
	err := g.Wait()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	// Show statistics
	showStat(statHistogram)
}

func pinger(ns string, m *dns.Msg, t time.Duration, statHistogram *stats.Histogram, done chan int) {
	c := dns.Client{
		Timeout: timeout,
	}
	ticker := time.NewTicker(t)
	for i := 1; count <= 0 || i <= count; i++ {
		select {
		case <-done:
			ticker.Stop()
			return
		case <-ticker.C:
			var buf bytes.Buffer
			res, rtt, err := c.Exchange(m, ns)
			if err != nil {
				fmt.Fprintf(&buf, "%v\n", time.Now())
				fmt.Fprintf(&buf, "Nameserver: %s\n", ns)
				fmt.Fprintf(&buf, "Exchange failed: %v (rtt: %v)\n", err, rtt)
				fmt.Println(buf.String())
				errorCount++
				continue
			}
			if res.Rcode != dns.RcodeSuccess {
				fmt.Fprintf(&buf, "%v\n", time.Now())
				fmt.Fprintf(&buf, "Nameserver: %s\n", ns)
				fmt.Fprintf(&buf, "Bad RCODE: %s (rtt: %v)\n", dns.RcodeToString[res.Rcode], rtt)
				fmt.Println(buf.String())
				errorCount++
				continue
			}
			rttMS := 1000. * rtt.Seconds()
			statHistogram.Record(rttMS)
			successCount++
			if !onlyError {
				fmt.Fprintf(&buf, "%v\n", time.Now())
				fmt.Fprintf(&buf, "Nameserver: %s\n", ns)
				for _, a := range res.Answer {
					fmt.Fprintln(&buf, a.String())
				}
				fmt.Println(buf.String())
			}
		}
	}
}

func showStat(statHistogram *stats.Histogram) {
	errorPerc := fmt.Sprintf("%.02f%%", 100.*float64(errorCount)/float64(errorCount+successCount))
	successPerc := fmt.Sprintf("%.02f%%", 100.*float64(successCount)/float64(errorCount+successCount))
	plural := "s" // 0 errors 1 error 2 errors...
	if errorCount == 1 {
		plural = ""
	}

	fmt.Printf("--- %s dns check statistics ---\n", name)
	fmt.Printf("%d error%s (%s) and %d success (%s) for %d nameservers.\n", errorCount, plural, errorPerc, successCount, successPerc, len(nameservers))
	res := statHistogram.Export()
	fmt.Printf("round-trip min/avg/max = %.3f/%.3f/%.3f ms\n", res.Min, res.Avg, res.Max)
}
