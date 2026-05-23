package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/sidecar"
)

var proxyConfig struct {
	appPort       int
	proxyPort     int
	rateLimit     int
	burst         int
	authMode      string
	apiKey        string
	circuitErrors float64
	circuitSleep  int
}

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Start the sidecar proxy",
	Long: `Starts the sidecar proxy with rate-limiting, auth, circuit breaking, 
and access logging capabilities. Forwards traffic to the application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.DefaultLogger

		config := sidecar.Config{
			ProxyPort: proxyConfig.proxyPort,
			AppPort:   proxyConfig.appPort,
			RateLimit: sidecar.RateLimitConfig{
				Enabled:     proxyConfig.rateLimit > 0,
				RequestsPer: proxyConfig.rateLimit,
				PerSeconds:  1,
				Burst:       proxyConfig.burst,
			},
			Auth: sidecar.AuthConfig{
				Enabled: proxyConfig.authMode != "",
				Mode:    proxyConfig.authMode,
				APIKey:  proxyConfig.apiKey,
			},
			Circuit: sidecar.CircuitConfig{
				Enabled:      proxyConfig.circuitErrors > 0,
				ErrorPercent: proxyConfig.circuitErrors,
				SleepWindow:  time.Duration(proxyConfig.circuitSleep) * time.Millisecond,
			},
			Logging: sidecar.LoggingConfig{
				Enabled:   true,
				AccessLog: true,
			},
		}

		targetURL := fmt.Sprintf("http://localhost:%d", proxyConfig.appPort)
		proxy, err := sidecar.NewProxy(targetURL, config)
		if err != nil {
			return fmt.Errorf("create proxy: %w", err)
		}

		chain := sidecar.BuildDefaultChain(config)
		handler := chain.Then(proxy.Handler())

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", proxyConfig.proxyPort),
			Handler: handler,
		}

		go func() {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
			<-sigCh
			logger.Info("Shutting down proxy...")
			server.Close()
		}()

		logger.Info("Sidecar proxy started: :%d → :%d", proxyConfig.proxyPort, proxyConfig.appPort)
		if proxyConfig.rateLimit > 0 {
			logger.Info("  Rate limit: %d req/s (burst: %d)", proxyConfig.rateLimit, proxyConfig.burst)
		}
		if proxyConfig.authMode != "" {
			logger.Info("  Auth mode: %s", proxyConfig.authMode)
		}
		if proxyConfig.circuitErrors > 0 {
			logger.Info("  Circuit breaker: error rate >= %.0f%%", proxyConfig.circuitErrors)
		}

		return server.ListenAndServe()
	},
}

func init() {
	proxyCmd.Flags().IntVarP(&proxyConfig.appPort, "app-port", "", 8080, "Application port")
	proxyCmd.Flags().IntVarP(&proxyConfig.proxyPort, "proxy-port", "p", 8443, "Proxy listening port")
	proxyCmd.Flags().IntVarP(&proxyConfig.rateLimit, "rate-limit", "r", 0, "Rate limit (requests/second)")
	proxyCmd.Flags().IntVarP(&proxyConfig.burst, "burst", "b", 1, "Rate limit burst")
	proxyCmd.Flags().StringVarP(&proxyConfig.authMode, "auth", "a", "", "Auth mode (apikey/jwt/basic)")
	proxyCmd.Flags().StringVarP(&proxyConfig.apiKey, "api-key", "k", "", "API key for auth")
	proxyCmd.Flags().Float64VarP(&proxyConfig.circuitErrors, "circuit-error", "", 0, "Circuit breaker error threshold (%)")
	proxyCmd.Flags().IntVarP(&proxyConfig.circuitSleep, "circuit-sleep", "", 5000, "Circuit breaker sleep (ms)")
	rootCmd.AddCommand(proxyCmd)
}