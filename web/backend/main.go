// MoonHub Web Console - Web-based chat and management interface
//
// Provides a web UI for chatting with MoonHub via the MoonHub WebSocket,
// with configuration management and gateway process control.
//
// Usage:
//
//	go build -o moonhub-web ./web/backend/
//	./moonhub-web [config.json]
//	./moonhub-web -public config.json

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/RealityLink-Tech/MoonHub/pkg/devices"
	"github.com/RealityLink-Tech/MoonHub/pkg/provisioning"
	"github.com/RealityLink-Tech/MoonHub/web/backend/api"
	"github.com/RealityLink-Tech/MoonHub/web/backend/launcherconfig"
	"github.com/RealityLink-Tech/MoonHub/web/backend/middleware"
	"github.com/RealityLink-Tech/MoonHub/web/backend/utils"
)

func main() {
	port := flag.String("port", "18800", "Port to listen on")
	public := flag.Bool("public", false, "Listen on all interfaces (0.0.0.0) instead of localhost only")
	noBrowser := flag.Bool("no-browser", false, "Do not auto-open browser on startup")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "MoonHub Launcher - A web-based configuration editor\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [config.json]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  config.json    Path to the configuration file (default: ~/.moonhub/config.json)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                          Use default config path\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ./config.json             Specify a config file\n", os.Args[0])
		fmt.Fprintf(
			os.Stderr,
			"  %s -public ./config.json     Allow access from other devices on the network\n",
			os.Args[0],
		)
	}
	flag.Parse()

	// Resolve config path
	configPath := utils.GetDefaultConfigPath()
	if flag.NArg() > 0 {
		configPath = flag.Arg(0)
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		log.Fatalf("Failed to resolve config path: %v", err)
	}
	err = utils.EnsureOnboarded(absPath)
	if err != nil {
		log.Printf("Warning: Failed to initialize MoonHub config automatically: %v", err)
	}

	var explicitPort bool
	var explicitPublic bool
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "port":
			explicitPort = true
		case "public":
			explicitPublic = true
		}
	})

	launcherPath := launcherconfig.PathForAppConfig(absPath)
	launcherCfg, err := launcherconfig.Load(launcherPath, launcherconfig.Default())
	if err != nil {
		log.Printf("Warning: Failed to load %s: %v", launcherPath, err)
		launcherCfg = launcherconfig.Default()
	}

	effectivePort := *port
	effectivePublic := *public
	if !explicitPort {
		effectivePort = strconv.Itoa(launcherCfg.Port)
	}
	if !explicitPublic {
		effectivePublic = launcherCfg.Public
	}

	portNum, err := strconv.Atoi(effectivePort)
	if err != nil || portNum < 1 || portNum > 65535 {
		if err == nil {
			err = errors.New("must be in range 1-65535")
		}
		log.Fatalf("Invalid port %q: %v", effectivePort, err)
	}

	// Determine listen address
	var addr string
	if effectivePublic {
		addr = "0.0.0.0:" + effectivePort
	} else {
		addr = "127.0.0.1:" + effectivePort
	}

	// Initialize Server components
	mux := http.NewServeMux()

	// Initialize device store and pairing manager
	deviceStoreDir := filepath.Dir(absPath)
	deviceStore, err := devices.NewDeviceStore(deviceStoreDir)
	if err != nil {
		log.Printf("Warning: Failed to initialize device store: %v", err)
		deviceStore, _ = devices.NewDeviceStore("") // fallback to default
	}
	pairingManager := devices.NewPairingManager(deviceStore)

	apiHandler := api.NewHandler(absPath, deviceStore, pairingManager)
	apiHandler.SetServerOptions(portNum, effectivePublic, explicitPublic, launcherCfg.AllowedCIDRs)

	// Device provisioning: SetProvisioningHandler must run before RegisterRoutes so routes are mounted.
	provisioningEnabled := os.Getenv("MOONHUB_PROVISIONING_ENABLED") == "1"
	if provisioningEnabled {
		provisioningConfigPath := filepath.Join(filepath.Dir(absPath), "provisioning.json")
		configStore, err := provisioning.NewJSONConfigStore(provisioningConfigPath)
		if err != nil {
			log.Printf("Warning: Failed to initialize provisioning config: %v", err)
		} else {
			deviceManager := provisioning.NewDeviceManager(configStore, provisioning.DeviceManagerOptions{
				AllowSystemControl: os.Getenv("MOONHUB_ALLOW_SYSTEM_CONTROL") == "1",
			})
			go deviceManager.Start(context.Background())
			provisioningHandler := api.NewProvisioningHandler(deviceManager)
			apiHandler.SetProvisioningHandler(provisioningHandler)
			log.Println("Device provisioning enabled")
		}
	}

	apiHandler.RegisterRoutes(mux)

	// Frontend Embedded Assets
	registerEmbedRoutes(mux)

	provisioningToken := os.Getenv("MOONHUB_PROVISIONING_TOKEN")
	if provisioningEnabled && effectivePublic && provisioningToken == "" {
		log.Printf("Warning: MOONHUB_PROVISIONING_ENABLED with public listen but MOONHUB_PROVISIONING_TOKEN is unset; provisioning API is reachable without a shared secret")
	}
	provAuthMux := middleware.ProvisioningAuth(provisioningToken, mux)

	accessControlledMux, err := middleware.IPAllowlist(launcherCfg.AllowedCIDRs, provAuthMux)
	if err != nil {
		log.Fatalf("Invalid allowed CIDR configuration: %v", err)
	}

	// Apply middleware stack (outermost first: recover → log → … → IP allowlist → provisioning auth → mux)
	handler := middleware.Recoverer(
		middleware.Logger(
			middleware.JSONContentType(accessControlledMux),
		),
	)

	// Print startup banner
	fmt.Print(utils.Banner)
	fmt.Println()
	fmt.Println("  Open the following URL in your browser:")
	fmt.Println()
	fmt.Printf("    >> http://localhost:%s <<\n", effectivePort)
	if effectivePublic {
		if ip := utils.GetLocalIP(); ip != "" {
			fmt.Printf("    >> http://%s:%s <<\n", ip, effectivePort)
		}
	}
	fmt.Println()

	// Auto-open browser
	if !*noBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			url := "http://localhost:" + effectivePort
			if err := utils.OpenBrowser(url); err != nil {
				log.Printf("Warning: Failed to auto-open browser: %v", err)
			}
		}()
	}

	// Auto-start gateway after backend starts listening.
	go func() {
		time.Sleep(1 * time.Second)
		apiHandler.TryAutoStartGateway()
	}()

	// Start the Server
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
