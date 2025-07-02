package main

import (
	"context"
	"fmt"
	"maps"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/urfave/cli/v3"
	etcd "go.etcd.io/etcd/client/v3"
)

type config struct {
	EtcdEndpoint   string `env:"SERVE_ETCD_ENDPOINT" envDefault:"localhost:2379"`
	EtcdUser       string `env:"SERVE_ETCD_USER"`
	EtcdPassword   string `env:"SERVE_ETCD_PASSWORD"`
	TargetIP       string `env:"SERVE_ETCD_TARGET_IP" envDefault:"127.0.0.1"`
	DomainTemplate string `env:"SERVE_DOMAIN_TEMPLATE"`
	CertResolver   string `env:"SERVE_CERT_RESOLVER" envDefault:"lecf"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("Error parsing environment variables: %+v\n", err)
		os.Exit(1)
	}

	cmd := &cli.Command{
		Name:  "serve",
		Usage: "Expose local apps via Traefik over etcd",
		Commands: []*cli.Command{
			{
				Name:      "run",
				Aliases:   []string{"start"},
				Usage:     "Add Traefik config for a local app",
				ArgsUsage: "<port>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "slug", Required: false, Usage: "Name of the app, e.g. myapp (auto-generated if not provided)"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return fmt.Errorf("exactly one argument (port) is required")
					}

					port := cmd.Args().Get(0)
					appName := cmd.String("slug")

					// Generate random app name if not provided
					if appName == "" {
						appName = generateRandomSlug()
						fmt.Printf("Generated app name: %s\n", appName)
					}

					// Normalize port: remove colon if present
					normalizedPort := strings.TrimPrefix(port, ":")

					activeServices, err := getActiveServices(cfg)
					if err != nil {
						return fmt.Errorf("could not get active services: %w", err)
					}
					for appName, port := range activeServices {
						if port == normalizedPort {
							return fmt.Errorf("port %s is already in use by app %s", normalizedPort, appName)
						}
					}

					domain := fmt.Sprintf(cfg.DomainTemplate, appName)

					if err := createTraefikConfig(cfg, appName, domain, port); err != nil {
						return fmt.Errorf("failed to create traefik config: %w", err)
					}

					fmt.Printf("Successfully configured %s to point to %s:%s\n", appName, cfg.TargetIP, normalizedPort)
					return nil
				},
			},
			{
				Name:      "stop",
				Usage:     "Remove a local app's Traefik config",
				ArgsUsage: "<slug|port>",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return fmt.Errorf("exactly one argument (slug or port) is required")
					}

					identifier := cmd.Args().Get(0)
					appName := identifier
					if isDigits(identifier) {
						// Assuming this is a port
						appName = findAppNameByPort(cfg, identifier)
					}

					if appName == "" {
						return fmt.Errorf("no app or port found for %s", identifier)
					}

					fmt.Printf("Removing traefik config for app: %s\n", appName)

					if err := removeTraefikConfig(cfg, appName); err != nil {
						return fmt.Errorf("failed to remove traefik config: %w", err)
					}

					fmt.Printf("Successfully stopped service for app: %s\n", appName)
					return nil
				},
			},
			{
				Name:    "status",
				Aliases: []string{"ls", "list"},
				Usage:   "Show currently active services",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					activeServices, err := getActiveServices(cfg)
					if err != nil {
						return fmt.Errorf("could not get active services: %w", err)
					}
					if len(activeServices) == 0 {
						fmt.Println("No active services found.")
						return nil
					}

					fmt.Printf("%-20s %-40s %s\n", "SLUG", "DOMAIN", "PORT")
					fmt.Printf("%-20s %-40s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 40), "----")
					for appName, port := range activeServices {
						var domainStr string
						domain := fmt.Sprintf(cfg.DomainTemplate, appName)
						domainStr = fmt.Sprintf("https://%s", domain)
						fmt.Printf("%-20s %-40s :%s\n",
							truncateString(appName, 20),
							truncateString(domainStr, 40),
							port)
					}
					return nil
				},
			},
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// getActiveServices scans etcd for traefik routers and services and returns a map of app_name -> port.
func getActiveServices(cfg config) (map[string]string, error) {
	client, err := createEtcdClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get all entries with prefix traefik/http/routers/
	resp, err := client.Get(ctx, "traefik/http/routers/", etcd.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to list etcd keys: %w", err)
	}

	services := make(map[string]string)

	// Extract unique router names
	routerNames := make(map[string]bool)
	for _, kv := range resp.Kvs {
		key := string(kv.Key)

		// Remove prefix traefik/http/routers/ and get first part
		afterPrefix := strings.TrimPrefix(key, "traefik/http/routers/")
		if afterPrefix == key { // prefix wasn't found
			continue
		}

		// Get router name (first element after prefix)
		parts := strings.Split(afterPrefix, "/")
		if len(parts) == 0 {
			continue
		}
		routerName := parts[0]
		routerNames[routerName] = true
	}

	// For each router, get the service and extract port
	for routerName := range routerNames {
		// Get service name
		svcResp, err := client.Get(ctx, fmt.Sprintf("traefik/http/routers/%s/service", routerName))
		if err != nil || len(svcResp.Kvs) == 0 {
			continue
		}
		serviceName := string(svcResp.Kvs[0].Value)

		// Get service URL
		serviceURLKey := fmt.Sprintf("traefik/http/services/%s/loadbalancer/servers/0/url", serviceName)
		serviceResp, err := client.Get(ctx, serviceURLKey)
		if err != nil || len(serviceResp.Kvs) == 0 {
			continue
		}

		serviceURL := string(serviceResp.Kvs[0].Value)
		u, _ := url.Parse(serviceURL)
		services[routerName] = u.Port()
	}

	return services, nil
}

func createEtcdClient(cfg config) (*etcd.Client, error) {
	clientCfg := etcd.Config{
		Endpoints:   []string{cfg.EtcdEndpoint},
		DialTimeout: 5 * time.Second,
	}

	if cfg.EtcdUser != "" && cfg.EtcdPassword != "" {
		clientCfg.Username = cfg.EtcdUser
		clientCfg.Password = cfg.EtcdPassword
	}

	return etcd.New(clientCfg)
}

func createTraefikConfig(cfg config, appName, domain string, port string) error {
	client, err := createEtcdClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Normalize port: remove colon if present, then ensure it has colon for URL
	normalizedPort := strings.TrimPrefix(port, ":")
	portWithColon := ":" + normalizedPort

	resourceName := appName
	serviceURL := fmt.Sprintf("http://%s%s", cfg.TargetIP, portWithColon)
	hostRule := fmt.Sprintf("Host(`%s`)", domain)

	// Create router configuration
	routerKeys := map[string]string{
		fmt.Sprintf("traefik/http/routers/%s/entrypoints", resourceName):      "https",
		fmt.Sprintf("traefik/http/routers/%s/tls", resourceName):              "true",
		fmt.Sprintf("traefik/http/routers/%s/tls/certresolver", resourceName): cfg.CertResolver,
		fmt.Sprintf("traefik/http/routers/%s/rule", resourceName):             hostRule,
		fmt.Sprintf("traefik/http/routers/%s/service", resourceName):          resourceName,
	}

	// Create service configuration
	serviceKeys := map[string]string{
		fmt.Sprintf("traefik/http/services/%s/loadbalancer/servers/0/url", resourceName): serviceURL,
	}

	// Combine all keys
	allKeys := make(map[string]string)
	maps.Copy(allKeys, routerKeys)
	maps.Copy(allKeys, serviceKeys)

	// Store all keys in etcd
	for key, value := range allKeys {
		_, err := client.Put(ctx, key, value)
		if err != nil {
			return fmt.Errorf("failed to put key %s: %w", key, err)
		}
	}

	return nil
}

func removeTraefikConfig(cfg config, appName string) error {
	client, err := createEtcdClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Delete router configuration
	routerPrefix := fmt.Sprintf("traefik/http/routers/%s/", appName)
	_, err = client.Delete(ctx, routerPrefix, etcd.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to delete router config: %w", err)
	}

	// Delete service configuration
	servicePrefix := fmt.Sprintf("traefik/http/services/%s/", appName)
	_, err = client.Delete(ctx, servicePrefix, etcd.WithPrefix())
	if err != nil {
		return fmt.Errorf("failed to delete service config: %w", err)
	}

	return nil
}

// generateRandomSlug creates a random 8-character alphanumeric string
func generateRandomSlug() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 8

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// findAppNameByPort searches etcd to find which app is using the specified port
func findAppNameByPort(cfg config, port string) string {
	client, err := createEtcdClient(cfg)
	if err != nil {
		return ""
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get all service URLs
	resp, err := client.Get(ctx, "traefik/http/services/", etcd.WithPrefix())
	if err != nil {
		return ""
	}

	targetURL := fmt.Sprintf(":%s", port)

	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		value := string(kv.Value)

		// Look for service URL keys like traefik/http/services/{app-name}/loadbalancer/servers/0/url
		if strings.HasSuffix(key, "/loadbalancer/servers/0/url") {
			if strings.HasSuffix(value, targetURL) {
				// Extract app name from key like traefik/http/services/myapp/loadbalancer/servers/0/url
				parts := strings.Split(key, "/")
				if len(parts) >= 5 {
					return parts[4] // app name is at index 4
				}
			}
		}
	}

	return ""
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
