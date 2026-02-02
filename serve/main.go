package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"
	etcd "go.etcd.io/etcd/client/v3"
)

type config struct {
	EtcdEndpoint   string
	EtcdUser       string
	EtcdPassword   string
	TargetIP       string
	DomainTemplate string
	CertResolver   string
	KeyPrefix      string
	SlugLength     int
}

func configFromCmd(cmd *cli.Command) config {
	root := cmd.Root()
	return config{
		EtcdEndpoint:   root.String("etcd-endpoint"),
		EtcdUser:       root.String("etcd-user"),
		EtcdPassword:   root.String("etcd-password"),
		TargetIP:       root.String("target-ip"),
		DomainTemplate: root.String("domain-template"),
		CertResolver:   root.String("cert-resolver"),
		KeyPrefix:      root.String("key-prefix"),
		SlugLength:     root.Int("slug-length"),
	}
}

func main() {
	configFilePath := os.Getenv("SERVE_CONFIG")
	if configFilePath == "" {
		configFilePath = "serve.yaml"
	}
	for i := 0; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--config" || os.Args[i] == "-c" {
			configFilePath = os.Args[i+1]
			break
		}
	}
	configFileSourcer := altsrc.NewStringPtrSourcer(&configFilePath)

	cmd := &cli.Command{
		Name:  "serve",
		Usage: "Expose local apps via Traefik over etcd",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "path to config file (YAML)",
				Value:       "serve.yaml",
				Destination: &configFilePath,
			},
			&cli.StringFlag{
				Name:  "etcd-endpoint",
				Usage: "etcd server endpoint",
				Value: "localhost:2379",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("SERVE_ETCD_ENDPOINT"),
					yaml.YAML("serve.etcd_endpoint", configFileSourcer),
				),
			},
			&cli.StringFlag{
				Name:  "etcd-user",
				Usage: "etcd username",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("SERVE_ETCD_USER"),
					yaml.YAML("serve.etcd_user", configFileSourcer),
				),
			},
			&cli.StringFlag{
				Name:  "etcd-password",
				Usage: "etcd password",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("SERVE_ETCD_PASSWORD"),
					yaml.YAML("serve.etcd_password", configFileSourcer),
				),
			},
			&cli.StringFlag{
				Name:  "target-ip",
				Usage: "Tailscale IP of local machine for Traefik to reach",
				Value: "127.0.0.1",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("SERVE_ETCD_TARGET_IP"),
					yaml.YAML("serve.target_ip", configFileSourcer),
				),
			},
			&cli.StringFlag{
				Name:  "domain-template",
				Usage: "domain template with %s for app name (e.g. %s.example.com)",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("SERVE_DOMAIN_TEMPLATE"),
					yaml.YAML("serve.domain_template", configFileSourcer),
				),
			},
			&cli.StringFlag{
				Name:  "cert-resolver",
				Usage: "Traefik cert resolver name",
				Value: "lecf",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("SERVE_CERT_RESOLVER"),
					yaml.YAML("serve.cert_resolver", configFileSourcer),
				),
			},
			&cli.StringFlag{
				Name:  "key-prefix",
				Usage: "prefix for router/service names in etcd (e.g. serve-myapp)",
				Value: "serve",
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("SERVE_KEY_PREFIX"),
					yaml.YAML("serve.key_prefix", configFileSourcer),
				),
			},
			&cli.IntFlag{
				Name:  "slug-length",
				Usage: "length of auto-generated slug (default 3)",
				Value: 3,
				Sources: cli.NewValueSourceChain(
					cli.EnvVar("SERVE_SLUG_LENGTH"),
					yaml.YAML("serve.slug_length", configFileSourcer),
				),
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "run",
				Aliases:   []string{"start"},
				Usage:     "Add Traefik config for a local app",
				ArgsUsage: "<port>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "slug", Required: false, Usage: "Name of the app, e.g. myapp (auto-generated if not provided)"},
					&cli.BoolFlag{Name: "detach", Aliases: []string{"d"}, Usage: "run in background (don't block; don't remove config on exit)"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					if cmd.NArg() != 1 {
						return fmt.Errorf("exactly one argument (port) is required")
					}

					cfg := configFromCmd(cmd)
					if cfg.DomainTemplate == "" {
						return fmt.Errorf("domain-template is required (set in config file, env SERVE_DOMAIN_TEMPLATE, or --domain-template)")
					}

					port := cmd.Args().Get(0)
					appName := cmd.String("slug")

					// Generate random app name if not provided
					if appName == "" {
						length := cfg.SlugLength
						if length < 1 {
							length = 3
						}
						appName = generateRandomSlug(length)
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
					resName := resourceName(cfg, appName)

					if err := createTraefikConfig(cfg, appName, domain, port); err != nil {
						return fmt.Errorf("failed to create traefik config: %w", err)
					}

					if cmd.Bool("detach") {
						fmt.Printf("Service available at https://%s (forwarding to :%s)\n", domain, normalizedPort)
						return nil
					}

					fmt.Printf("Serving at https://%s (forwarding to :%s). Press Ctrl+C to stop and remove from Traefik.\n", domain, normalizedPort)
					sigCh := make(chan os.Signal, 1)
					signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
					<-sigCh
					if err := removeTraefikConfig(cfg, resName); err != nil {
						return fmt.Errorf("failed to remove traefik config on exit: %w", err)
					}
					fmt.Println("Removed from Traefik.")
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

					cfg := configFromCmd(cmd)
					identifier := cmd.Args().Get(0)
					var resourceNameForDelete string
					if isDigits(identifier) {
						// Port: findAppNameByPort returns full etcd resource name (e.g. serve-myapp)
						resourceNameForDelete = findAppNameByPort(cfg, identifier)
					} else {
						// Slug: build full resource name
						resourceNameForDelete = resourceName(cfg, identifier)
					}

					if resourceNameForDelete == "" {
						return fmt.Errorf("no app or port found for %s", identifier)
					}

					slugDisplay := slugFromResourceName(cfg, resourceNameForDelete)
					fmt.Printf("Removing traefik config for app: %s\n", slugDisplay)

					if err := removeTraefikConfig(cfg, resourceNameForDelete); err != nil {
						return fmt.Errorf("failed to remove traefik config: %w", err)
					}

					fmt.Printf("Successfully stopped service for app: %s\n", slugDisplay)
					return nil
				},
			},
			{
				Name:      "clean",
				Aliases:   []string{"reset"},
				Usage:     "Remove all Traefik config for services managed by this utility (key prefix)",
				ArgsUsage: " ",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg := configFromCmd(cmd)
					activeServices, err := getActiveServices(cfg)
					if err != nil {
						return fmt.Errorf("could not get active services: %w", err)
					}
					if len(activeServices) == 0 {
						fmt.Println("No active services to clean.")
						return nil
					}
					for slug := range activeServices {
						if err := removeTraefikConfig(cfg, resourceName(cfg, slug)); err != nil {
							return fmt.Errorf("failed to remove %s: %w", slug, err)
						}
						fmt.Printf("Stopped %s\n", slug)
					}
					fmt.Printf("Cleaned %d service(s).\n", len(activeServices))
					return nil
				},
			},
			{
				Name:    "status",
				Aliases: []string{"ls", "list"},
				Usage:   "Show currently active services",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					cfg := configFromCmd(cmd)
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
		// Only include routers that match our key prefix (if set)
		if cfg.KeyPrefix != "" && !strings.HasPrefix(routerName, cfg.KeyPrefix+"-") {
			continue
		}
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
		slug := slugFromResourceName(cfg, routerName)
		services[slug] = u.Port()
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

	resName := resourceName(cfg, appName)
	serviceURL := fmt.Sprintf("http://%s%s", cfg.TargetIP, portWithColon)
	hostRule := fmt.Sprintf("Host(`%s`)", domain)

	// Create router configuration
	routerKeys := map[string]string{
		fmt.Sprintf("traefik/http/routers/%s/entrypoints", resName):      "https",
		fmt.Sprintf("traefik/http/routers/%s/tls", resName):              "true",
		fmt.Sprintf("traefik/http/routers/%s/tls/certresolver", resName): cfg.CertResolver,
		fmt.Sprintf("traefik/http/routers/%s/rule", resName):             hostRule,
		fmt.Sprintf("traefik/http/routers/%s/service", resName):          resName,
	}

	// Create service configuration
	serviceKeys := map[string]string{
		fmt.Sprintf("traefik/http/services/%s/loadbalancer/servers/0/url", resName): serviceURL,
	}

	// Store keys in etcd: service first, then routers (deterministic order)
	for key, value := range serviceKeys {
		_, err := client.Put(ctx, key, value)
		if err != nil {
			return fmt.Errorf("failed to put key %s: %w", key, err)
		}
	}
	for key, value := range routerKeys {
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

// generateRandomSlug creates a random alphanumeric string of the given length
func generateRandomSlug(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	if length < 1 {
		length = 3
	}
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

// resourceName returns the etcd/Traefik resource name (router/service name). If KeyPrefix is set, it is prefix-slug; otherwise slug only.
func resourceName(cfg config, slug string) string {
	if cfg.KeyPrefix == "" {
		return slug
	}
	return cfg.KeyPrefix + "-" + slug
}

// slugFromResourceName returns the slug from a resource name (strips KeyPrefix if present).
func slugFromResourceName(cfg config, name string) string {
	if cfg.KeyPrefix == "" {
		return name
	}
	prefix := cfg.KeyPrefix + "-"
	name, _ = strings.CutPrefix(name, prefix)
	return name
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
