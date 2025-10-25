package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Commands: []*cli.Command{
			{
				Name:    "resolve",
				Aliases: []string{"r"},
				Usage:   "resolve domains from a file",
				Action:  resolveDomainsAction,
			},
			{
				Name:    "analyze",
				Aliases: []string{"a"},
				Usage:   "analyze domain patterns and statistics",
				Action:  analyzeDomainsAction,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

type DomainResult struct {
	Domain      string
	IPv4        []string
	IPv6        []string
	Error       string
	ResolveTime time.Duration
}

type IPRange struct {
	Network string
	CIDR    string
	Count   int
}

func resolveDomainsAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("usage: resolve <domains.txt>")
	}

	filename := cmd.Args().First()
	domains, err := readDomainsFromFile(filename)
	if err != nil {
		return fmt.Errorf("error reading domains file: %v", err)
	}

	fmt.Printf("Resolving %d domains...\n", len(domains))

	results := resolveDomains(domains)

	// Print individual results
	printResults(results)

	// Analyze IP ranges
	analyzeIPRanges(results)

	return nil
}

func analyzeDomainsAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("usage: analyze <domains.txt>")
	}

	filename := cmd.Args().First()
	domains, err := readDomainsFromFile(filename)
	if err != nil {
		return fmt.Errorf("error reading domains file: %v", err)
	}

	fmt.Printf("Analyzing %d domains...\n", len(domains))

	// Basic domain analysis
	domainStats := analyzeDomainPatterns(domains)
	printDomainAnalysis(domainStats)

	return nil
}

func readDomainsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var domains []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			domains = append(domains, line)
		}
	}

	return domains, scanner.Err()
}

func resolveDomains(domains []string) []DomainResult {
	results := make([]DomainResult, len(domains))
	var wg sync.WaitGroup

	for i, domain := range domains {
		wg.Add(1)
		go func(index int, d string) {
			defer wg.Done()
			start := time.Now()

			result := DomainResult{
				Domain: d,
			}

			// Resolve IPv4
			ipv4s, err := net.LookupIP(d)
			if err != nil {
				result.Error = err.Error()
			} else {
				for _, ip := range ipv4s {
					if ip.To4() != nil {
						result.IPv4 = append(result.IPv4, ip.String())
					} else {
						result.IPv6 = append(result.IPv6, ip.String())
					}
				}
			}

			result.ResolveTime = time.Since(start)
			results[index] = result
		}(i, domain)
	}

	wg.Wait()
	return results
}

func printResults(results []DomainResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("DOMAIN RESOLUTION RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	for _, result := range results {
		fmt.Printf("\nDomain: %s\n", result.Domain)
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		} else {
			if len(result.IPv4) > 0 {
				fmt.Printf("  IPv4: %s\n", strings.Join(result.IPv4, ", "))
			}
			if len(result.IPv6) > 0 {
				fmt.Printf("  IPv6: %s\n", strings.Join(result.IPv6, ", "))
			}
		}
		fmt.Printf("  Resolve time: %v\n", result.ResolveTime)
	}
}

func analyzeIPRanges(results []DomainResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("IP RANGE ANALYSIS")
	fmt.Println(strings.Repeat("=", 80))

	allIPs := make(map[string]int)

	// Collect all unique IPs
	for _, result := range results {
		if result.Error == "" {
			for _, ip := range result.IPv4 {
				allIPs[ip]++
			}
			for _, ip := range result.IPv6 {
				allIPs[ip]++
			}
		}
	}

	// Find common subnets
	subnets := findCommonSubnets(allIPs)

	fmt.Printf("\nTotal unique IPs found: %d\n", len(allIPs))
	fmt.Printf("Common subnets:\n")

	for _, subnet := range subnets {
		fmt.Printf("  %s (%s) - %d IPs\n", subnet.Network, subnet.CIDR, subnet.Count)
	}

	// Show IPs by frequency
	fmt.Println("\nIPs by frequency:")
	var ipFreq []struct {
		IP   string
		Freq int
	}

	for ip, freq := range allIPs {
		ipFreq = append(ipFreq, struct {
			IP   string
			Freq int
		}{ip, freq})
	}

	sort.Slice(ipFreq, func(i, j int) bool {
		return ipFreq[i].Freq > ipFreq[j].Freq
	})

	for _, item := range ipFreq {
		if item.Freq > 1 {
			fmt.Printf("  %s: %d domains\n", item.IP, item.Freq)
		}
	}
}

func findCommonSubnets(ips map[string]int) []IPRange {
	subnets := make(map[string]int)

	for ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		// Check common subnet sizes
		for _, cidr := range []string{"/24", "/16", "/8"} {
			_, network, err := net.ParseCIDR(ipStr + cidr)
			if err != nil {
				continue
			}

			networkStr := network.String()
			subnets[networkStr]++
		}
	}

	var ranges []IPRange
	for network, count := range subnets {
		if count > 1 {
			ranges = append(ranges, IPRange{
				Network: network,
				CIDR:    network,
				Count:   count,
			})
		}
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Count > ranges[j].Count
	})

	return ranges
}

type DomainStats struct {
	TotalDomains   int
	TLDs           map[string]int
	Subdomains     map[string]int
	CommonPatterns []string
	AverageLength  float64
}

func analyzeDomainPatterns(domains []string) DomainStats {
	stats := DomainStats{
		TLDs:       make(map[string]int),
		Subdomains: make(map[string]int),
	}

	totalLength := 0
	for _, domain := range domains {
		stats.TotalDomains++
		totalLength += len(domain)

		// Extract TLD (last part after final dot)
		parts := strings.Split(domain, ".")
		if len(parts) > 1 {
			tld := parts[len(parts)-1]
			stats.TLDs[tld]++
		}

		// Extract subdomains (everything before the final dot)
		if len(parts) > 2 {
			subdomain := strings.Join(parts[:len(parts)-2], ".")
			stats.Subdomains[subdomain]++
		}
	}

	stats.AverageLength = float64(totalLength) / float64(stats.TotalDomains)

	// Find common patterns
	for tld, count := range stats.TLDs {
		if count > 1 {
			stats.CommonPatterns = append(stats.CommonPatterns, fmt.Sprintf("%s (TLD): %d domains", tld, count))
		}
	}

	return stats
}

func printDomainAnalysis(stats DomainStats) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("DOMAIN ANALYSIS")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("\nTotal domains: %d\n", stats.TotalDomains)
	fmt.Printf("Average domain length: %.2f characters\n", stats.AverageLength)

	fmt.Println("\nTop TLDs:")
	var tldFreq []struct {
		TLD   string
		Count int
	}
	for tld, count := range stats.TLDs {
		tldFreq = append(tldFreq, struct {
			TLD   string
			Count int
		}{tld, count})
	}
	sort.Slice(tldFreq, func(i, j int) bool {
		return tldFreq[i].Count > tldFreq[j].Count
	})

	for i, item := range tldFreq {
		if i >= 10 { // Show top 10
			break
		}
		fmt.Printf("  %s: %d domains\n", item.TLD, item.Count)
	}

	fmt.Println("\nCommon patterns:")
	for _, pattern := range stats.CommonPatterns {
		fmt.Printf("  %s\n", pattern)
	}
}
