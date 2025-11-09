package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
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
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "output file for subnets and frequent IPs",
					},
				},
			},
			{
				Name:    "analyze",
				Aliases: []string{"a"},
				Usage:   "analyze domain patterns and statistics",
				Action:  analyzeDomainsAction,
			},
			{
				Name:    "fronting",
				Aliases: []string{"f"},
				Usage:   "check if domain fronting is possible between two domains",
				Action:  domainFrontingAction,
			},
			{
				Name:    "check",
				Aliases: []string{"c"},
				Usage:   "check IP subnets against popular websites for geo info and ownership",
				Action:  checkIPsAction,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "output file for IP analysis results",
					},
				},
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

type IPInfo struct {
	IP          string
	Country     string
	CountryCode string
	Region      string
	City        string
	ISP         string
	Org         string
	ASN         string
	Error       string
}

type IPCheckResult struct {
	IP          string
	Country     string
	CountryCode string
	Region      string
	City        string
	ISP         string
	Org         string
	ASN         string
	Error       string
	CheckTime   time.Duration
}

type GeoGroup struct {
	Country     string
	CountryCode string
	Region      string
	City        string
	IPs         []IPCheckResult
	Count       int
}

type OwnerGroup struct {
	Org   string
	ISP   string
	ASN   string
	IPs   []IPCheckResult
	Count int
}

func resolveDomainsAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("usage: resolve <domains.txt> [--output|-o output.txt]")
	}

	filename := cmd.Args().First()
	outputFile := cmd.String("output")

	domains, err := readDomainsFromFile(filename)
	if err != nil {
		return fmt.Errorf("error reading domains file: %v", err)
	}

	fmt.Printf("Resolving %d domains...\n", len(domains))

	results := resolveDomains(domains)

	// Print individual results
	printResults(results)

	// Analyze IP ranges
	analyzeIPRanges(results, outputFile)

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

func checkIPsAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 1 {
		return fmt.Errorf("usage: check <ips.txt> [--output|-o output.txt]")
	}

	filename := cmd.Args().First()
	outputFile := cmd.String("output")

	ips, err := readIPsFromFile(filename)
	if err != nil {
		return fmt.Errorf("error reading IPs file: %v", err)
	}

	fmt.Printf("Checking %d IPs/subnets...\n", len(ips))

	results := checkIPs(ips)

	// Print individual results
	printIPCheckResults(results)

	// Group by geo info and owner
	groupByGeoAndOwner(results, outputFile)

	return nil
}

func readIPsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ips []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			ips = append(ips, line)
		}
	}

	return ips, scanner.Err()
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

func checkIPs(ips []string) []IPCheckResult {
	var allResults []IPCheckResult
	var wg sync.WaitGroup

	for _, ip := range ips {
		// Check if it's a CIDR range
		if strings.Contains(ip, "/") {
			// Extract sample IPs from the CIDR range
			sampleIPs := getSampleIPsFromCIDR(ip)
			for _, sampleIP := range sampleIPs {
				wg.Add(1)
				go func(ipStr string, originalRange string) {
					defer wg.Done()
					start := time.Now()

					result := IPCheckResult{
						IP: originalRange + " (sample: " + ipStr + ")",
					}

					// Get IP info from multiple sources
					ipInfo := getIPInfo(ipStr)
					if ipInfo.Error != "" {
						result.Error = ipInfo.Error
					} else {
						result.Country = ipInfo.Country
						result.CountryCode = ipInfo.CountryCode
						result.Region = ipInfo.Region
						result.City = ipInfo.City
						result.ISP = ipInfo.ISP
						result.Org = ipInfo.Org
						result.ASN = ipInfo.ASN
					}

					result.CheckTime = time.Since(start)
					allResults = append(allResults, result)
				}(sampleIP, ip)
			}
		} else {
			// Single IP address
			wg.Add(1)
			go func(ipStr string) {
				defer wg.Done()
				start := time.Now()

				result := IPCheckResult{
					IP: ipStr,
				}

				// Get IP info from multiple sources
				ipInfo := getIPInfo(ipStr)
				if ipInfo.Error != "" {
					result.Error = ipInfo.Error
				} else {
					result.Country = ipInfo.Country
					result.CountryCode = ipInfo.CountryCode
					result.Region = ipInfo.Region
					result.City = ipInfo.City
					result.ISP = ipInfo.ISP
					result.Org = ipInfo.Org
					result.ASN = ipInfo.ASN
				}

				result.CheckTime = time.Since(start)
				allResults = append(allResults, result)
			}(ip)
		}
	}

	wg.Wait()
	return allResults
}

func getIPInfo(ipStr string) IPInfo {
	// Try ipapi.co first (free, no API key required)
	info := getIPInfoFromAPI(ipStr, "http://ipapi.co/"+ipStr+"/json/")
	if info.Error == "" {
		return info
	}

	// Fallback to ip-api.com
	info = getIPInfoFromAPI(ipStr, "http://ip-api.com/json/"+ipStr)
	if info.Error == "" {
		return info
	}

	// Fallback to ipinfo.io
	info = getIPInfoFromAPI(ipStr, "https://ipinfo.io/"+ipStr+"/json")
	return info
}

func getIPInfoFromAPI(ipStr, url string) IPInfo {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return IPInfo{IP: ipStr, Error: err.Error()}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return IPInfo{IP: ipStr, Error: err.Error()}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return IPInfo{IP: ipStr, Error: err.Error()}
	}

	// Check for error in response
	if status, ok := data["status"].(string); ok && status == "fail" {
		if message, ok := data["message"].(string); ok {
			return IPInfo{IP: ipStr, Error: message}
		}
	}

	info := IPInfo{IP: ipStr}

	// Parse common fields from different APIs
	if country, ok := data["country"].(string); ok {
		info.Country = country
	}
	if countryCode, ok := data["country_code"].(string); ok {
		info.CountryCode = countryCode
	}
	if region, ok := data["region"].(string); ok {
		info.Region = region
	}
	if city, ok := data["city"].(string); ok {
		info.City = city
	}
	if isp, ok := data["isp"].(string); ok {
		info.ISP = isp
	}
	if org, ok := data["org"].(string); ok {
		info.Org = org
	}
	if asn, ok := data["asn"].(string); ok {
		info.ASN = asn
	}

	// Handle different field names for different APIs
	if info.Country == "" {
		if country, ok := data["country_name"].(string); ok {
			info.Country = country
		}
	}
	if info.CountryCode == "" {
		if countryCode, ok := data["countryCode"].(string); ok {
			info.CountryCode = countryCode
		}
	}
	if info.Region == "" {
		if region, ok := data["regionName"].(string); ok {
			info.Region = region
		}
	}
	if info.City == "" {
		if city, ok := data["cityName"].(string); ok {
			info.City = city
		}
	}
	if info.ISP == "" {
		if isp, ok := data["isp"].(string); ok {
			info.ISP = isp
		}
	}
	if info.Org == "" {
		if org, ok := data["organization"].(string); ok {
			info.Org = org
		}
	}

	return info
}

func getSampleIPsFromCIDR(cidr string) []string {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return []string{cidr} // Return original if parsing fails
	}

	var sampleIPs []string

	// Get the network address and broadcast address
	ip := ipNet.IP.To4()
	if ip == nil {
		// IPv6
		ip = ipNet.IP.To16()
		if ip == nil {
			return []string{cidr}
		}
	}

	// For small ranges, sample a few IPs
	ones, bits := ipNet.Mask.Size()
	hostBits := bits - ones
	totalHosts := 1 << hostBits

	// Limit to reasonable number of samples
	maxSamples := 5
	if totalHosts < maxSamples {
		maxSamples = totalHosts
	}

	// For very large ranges, limit samples even more
	if totalHosts > 1000 {
		maxSamples = 3
	}

	// Sample IPs from the range
	var step int
	if maxSamples > 0 {
		step = totalHosts / maxSamples
	}
	if step == 0 {
		step = 1
	}

	for i := 0; i < maxSamples && i*step < totalHosts; i++ {
		// Calculate offset from network address
		offset := i * step

		// Add offset to network IP
		sampleIP := make(net.IP, len(ip))
		copy(sampleIP, ip)

		// Add the offset
		for j := len(sampleIP) - 1; j >= 0 && offset > 0; j-- {
			carry := offset & 0xFF
			sampleIP[j] += byte(carry)
			offset >>= 8
		}

		// Make sure we don't exceed the broadcast address
		if ipNet.Contains(sampleIP) {
			sampleIPs = append(sampleIPs, sampleIP.String())
		}
	}

	// If we couldn't generate samples, try the network address itself
	if len(sampleIPs) == 0 {
		sampleIPs = append(sampleIPs, ipNet.IP.String())
	}

	return sampleIPs
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

func printIPCheckResults(results []IPCheckResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("IP CHECK RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	for _, result := range results {
		fmt.Printf("\nIP: %s\n", result.IP)
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		} else {
			fmt.Printf("  Country: %s (%s)\n", result.Country, result.CountryCode)
			fmt.Printf("  Region: %s\n", result.Region)
			fmt.Printf("  City: %s\n", result.City)
			fmt.Printf("  ISP: %s\n", result.ISP)
			fmt.Printf("  Organization: %s\n", result.Org)
			fmt.Printf("  ASN: %s\n", result.ASN)
		}
		fmt.Printf("  Check time: %v\n", result.CheckTime)
	}
}

func groupByGeoAndOwner(results []IPCheckResult, outputFile string) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("GROUPED ANALYSIS")
	fmt.Println(strings.Repeat("=", 80))

	// Group by geographic location
	geoGroups := make(map[string]*GeoGroup)
	ownerGroups := make(map[string]*OwnerGroup)

	for _, result := range results {
		if result.Error != "" {
			continue
		}

		// Group by geo location
		geoKey := fmt.Sprintf("%s|%s|%s", result.Country, result.Region, result.City)
		if geoGroups[geoKey] == nil {
			geoGroups[geoKey] = &GeoGroup{
				Country:     result.Country,
				CountryCode: result.CountryCode,
				Region:      result.Region,
				City:        result.City,
				IPs:         []IPCheckResult{},
			}
		}
		geoGroups[geoKey].IPs = append(geoGroups[geoKey].IPs, result)
		geoGroups[geoKey].Count++

		// Group by owner
		ownerKey := fmt.Sprintf("%s|%s|%s", result.Org, result.ISP, result.ASN)
		if ownerGroups[ownerKey] == nil {
			ownerGroups[ownerKey] = &OwnerGroup{
				Org: result.Org,
				ISP: result.ISP,
				ASN: result.ASN,
				IPs: []IPCheckResult{},
			}
		}
		ownerGroups[ownerKey].IPs = append(ownerGroups[ownerKey].IPs, result)
		ownerGroups[ownerKey].Count++
	}

	// Print geographic groups
	fmt.Println("\nBy Geographic Location:")
	var geoList []*GeoGroup
	for _, group := range geoGroups {
		geoList = append(geoList, group)
	}
	sort.Slice(geoList, func(i, j int) bool {
		return geoList[i].Count > geoList[j].Count
	})

	for _, group := range geoList {
		fmt.Printf("\n  %s, %s, %s (%d IPs)\n", group.City, group.Region, group.Country, group.Count)
		for _, ip := range group.IPs {
			fmt.Printf("    %s\n", ip.IP)
		}
	}

	// Print owner groups
	fmt.Println("\nBy Owner/ISP:")
	var ownerList []*OwnerGroup
	for _, group := range ownerGroups {
		ownerList = append(ownerList, group)
	}
	sort.Slice(ownerList, func(i, j int) bool {
		return ownerList[i].Count > ownerList[j].Count
	})

	for _, group := range ownerList {
		fmt.Printf("\n  %s / %s / %s (%d IPs)\n", group.Org, group.ISP, group.ASN, group.Count)
		for _, ip := range group.IPs {
			fmt.Printf("    %s\n", ip.IP)
		}
	}

	// Write to output file if specified
	if outputFile != "" {
		writeIPCheckAnalysisToFile(geoList, ownerList, outputFile)
	}
}

func writeIPCheckAnalysisToFile(geoGroups []*GeoGroup, ownerGroups []*OwnerGroup, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write geographic groups
	writer.WriteString("# Geographic Groups\n")
	for _, group := range geoGroups {
		writer.WriteString(fmt.Sprintf("\n## %s, %s, %s (%d IPs)\n", group.City, group.Region, group.Country, group.Count))
		for _, ip := range group.IPs {
			writer.WriteString(fmt.Sprintf("%s\n", ip.IP))
		}
	}

	// Write owner groups
	writer.WriteString("\n\n# Owner/ISP Groups\n")
	for _, group := range ownerGroups {
		writer.WriteString(fmt.Sprintf("\n## %s / %s / %s (%d IPs)\n", group.Org, group.ISP, group.ASN, group.Count))
		for _, ip := range group.IPs {
			writer.WriteString(fmt.Sprintf("%s\n", ip.IP))
		}
	}

	fmt.Printf("\nIP analysis written to: %s\n", filename)
}

func analyzeIPRanges(results []DomainResult, outputFile string) {
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

	// Write to output file if specified
	if outputFile != "" {
		writeIPAnalysisToFile(subnets, ipFreq, outputFile)
	}
}

func writeIPAnalysisToFile(subnets []IPRange, ipFreq []struct {
	IP   string
	Freq int
}, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write subnets
	writer.WriteString("# Common Subnets\n")
	for _, subnet := range subnets {
		writer.WriteString(fmt.Sprintf("%s\n", subnet.Network))
	}

	// Write frequent IPs
	writer.WriteString("\n# Frequent IPs (appearing in multiple domains)\n")
	for _, item := range ipFreq {
		if item.Freq > 1 {
			writer.WriteString(fmt.Sprintf("%s\n", item.IP))
		}
	}

	fmt.Printf("\nIP analysis written to: %s\n", filename)
}

func findCommonSubnets(ips map[string]int) []IPRange {
	subnets := make(map[string]int)

	for ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		// Check subnet sizes, excluding /8 (top-level) - only /24 and /16
		for _, cidr := range []string{"/24", "/16"} {
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

type FrontingResult struct {
	YourDomain   string
	TargetDomain string
	Possible     bool
	Reason       string
	SNIResponse  string
	Error        string
	TestDuration time.Duration
}

func domainFrontingAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() < 2 {
		return fmt.Errorf("usage: fronting <your-domain> <target-domain>")
	}

	yourDomain := cmd.Args().Get(0)
	targetDomain := cmd.Args().Get(1)

	fmt.Printf("Testing domain fronting: %s -> %s\n", yourDomain, targetDomain)

	result := testDomainFronting(yourDomain, targetDomain)
	printFrontingResult(result)

	return nil
}

func testDomainFronting(yourDomain, targetDomain string) FrontingResult {
	start := time.Now()
	result := FrontingResult{
		YourDomain:   yourDomain,
		TargetDomain: targetDomain,
	}

	// First, resolve both domains to get their IPs
	yourIPs, err := net.LookupIP(yourDomain)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to resolve your domain: %v", err)
		return result
	}

	targetIPs, err := net.LookupIP(targetDomain)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to resolve target domain: %v", err)
		return result
	}

	// Check if domains share the same IP (common for CDNs)
	yourIPSet := make(map[string]bool)
	for _, ip := range yourIPs {
		yourIPSet[ip.String()] = true
	}

	sharedIPs := make([]string, 0)
	for _, ip := range targetIPs {
		if yourIPSet[ip.String()] {
			sharedIPs = append(sharedIPs, ip.String())
		}
	}

	if len(sharedIPs) == 0 {
		result.Possible = false
		result.Reason = "No shared IP addresses between domains"
		result.TestDuration = time.Since(start)
		return result
	}

	// Test SNI-based domain fronting
	// We'll try to connect to the target domain's IP but use your domain in SNI
	testIP := sharedIPs[0]

	// Create a custom TLS config that uses your domain in SNI
	config := &tls.Config{
		ServerName:         yourDomain,
		InsecureSkipVerify: true, // We're testing, so skip cert verification
	}

	// Try to establish TLS connection with SNI fronting
	conn, err := tls.DialWithDialer(&net.Dialer{
		Timeout: 10 * time.Second,
	}, "tcp", testIP+":443", config)

	if err != nil {
		result.Possible = false
		result.Reason = fmt.Sprintf("TLS connection failed: %v", err)
		result.TestDuration = time.Since(start)
		return result
	}
	defer conn.Close()

	// Check what certificate we actually received
	state := conn.ConnectionState()
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		result.SNIResponse = cert.Subject.CommonName

		// Check if the certificate is for the target domain or your domain
		certDomains := cert.DNSNames
		certDomains = append(certDomains, cert.Subject.CommonName)

		yourDomainMatch := false
		targetDomainMatch := false

		for _, domain := range certDomains {
			if domain == yourDomain || strings.HasSuffix(domain, "."+yourDomain) {
				yourDomainMatch = true
			}
			if domain == targetDomain || strings.HasSuffix(domain, "."+targetDomain) {
				targetDomainMatch = true
			}
		}

		if yourDomainMatch && !targetDomainMatch {
			result.Possible = true
			result.Reason = "SNI fronting appears to work - received certificate for your domain"
		} else if targetDomainMatch {
			result.Possible = false
			result.Reason = "Server correctly routes to target domain based on SNI"
		} else {
			result.Possible = false
			result.Reason = "Certificate doesn't match either domain"
		}
	} else {
		result.Possible = false
		result.Reason = "No certificate received"
	}

	result.TestDuration = time.Since(start)
	return result
}

func printFrontingResult(result FrontingResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("DOMAIN FRONTING TEST RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("\nYour Domain: %s\n", result.YourDomain)
	fmt.Printf("Target Domain: %s\n", result.TargetDomain)

	if result.Error != "" {
		fmt.Printf("Error: %s\n", result.Error)
		return
	}

	fmt.Printf("Domain Fronting Possible: %t\n", result.Possible)
	fmt.Printf("Reason: %s\n", result.Reason)

	if result.SNIResponse != "" {
		fmt.Printf("Certificate Subject: %s\n", result.SNIResponse)
	}

	fmt.Printf("Test Duration: %v\n", result.TestDuration)

	if result.Possible {
		fmt.Println("\n⚠️  WARNING: Domain fronting appears to be possible!")
		fmt.Println("   This could potentially be used to bypass domain-based filtering.")
	} else {
		fmt.Println("\n✅ Domain fronting does not appear to be possible.")
	}
}
