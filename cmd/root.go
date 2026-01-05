package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	awsfactory "github.com/keanuharrell/a9s/internal/aws"
	"github.com/keanuharrell/a9s/internal/config"
	"github.com/keanuharrell/a9s/internal/core"
	"github.com/keanuharrell/a9s/internal/hooks"
	"github.com/keanuharrell/a9s/internal/hooks/builtin"
	"github.com/keanuharrell/a9s/internal/registry"
	"github.com/keanuharrell/a9s/internal/services/ec2"
	"github.com/keanuharrell/a9s/internal/services/iam"
	"github.com/keanuharrell/a9s/internal/services/lambda"
	"github.com/keanuharrell/a9s/internal/services/s3"
	"github.com/keanuharrell/a9s/internal/tui"
)

var (
	// Version is set via ldflags during build
	Version = "dev"
	// BuildTime is set via ldflags during build
	BuildTime = "unknown"

	// CLI flags
	outputFormat string
	awsProfile   string
	awsRegion    string
	dryRun       bool
	configFile   string
	verbose      bool
)

var rootCmd = &cobra.Command{
	Use:   "a9s",
	Short: "Interactive Terminal UI for AWS infrastructure management",
	Long: `a9s is the k9s for AWS - an interactive Terminal UI that simplifies AWS infrastructure management.

It provides commands for:
- EC2 instance management and monitoring
- IAM security auditing and compliance checks
- S3 bucket cleanup and optimization
- Lambda function management
- Interactive Terminal UI (TUI) for real-time management

Usage:
  a9s          Launch interactive TUI (default)
  a9s tui      Launch interactive TUI explicitly
  a9s [cmd]    Run specific CLI commands`,
	Version: Version,
	Run: func(_ *cobra.Command, _ []string) {
		if err := runTUI(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// GetRootCommand returns the root command for external use (e.g., man page generation)
func GetRootCommand() *cobra.Command {
	return rootCmd
}

// =============================================================================
// TUI Initialization
// =============================================================================

func runTUI() error {
	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply CLI flag overrides
	applyFlagOverrides(cfg)

	// Create AWS client factory
	awsCfg := cfg.AWS.ToCore()
	factory, err := awsfactory.NewClientFactory(awsCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS: %w", err)
	}

	// Create event dispatcher with hooks
	dispatcher := createDispatcher(cfg)

	// Create registry
	reg := registry.New()

	// Register services
	if err := registerServices(reg, factory, cfg, dispatcher); err != nil {
		return fmt.Errorf("failed to register services: %w", err)
	}

	// Create and run TUI
	app := tui.NewApp(reg, cfg, dispatcher)
	app.SetFactory(factory)

	program := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err = program.Run()
	if err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	// Cleanup
	cleanupDispatcher(dispatcher)
	for _, svc := range reg.ListServices() {
		_ = svc.Close()
	}

	return nil
}

// =============================================================================
// Event Dispatcher Setup
// =============================================================================

// createDispatcher creates and configures the event dispatcher.
func createDispatcher(cfg *config.Config) *hooks.Dispatcher {
	dispatcher := hooks.NewDispatcher()

	// Add recovery middleware to prevent hook panics from crashing the app
	dispatcher.Use(&hooks.RecoveryMiddleware{
		OnPanic: func(hook string, r any) {
			fmt.Fprintf(os.Stderr, "Hook panic in %s: %v\n", hook, r)
		},
	})

	// Add logging hook if verbose mode or configured
	if verbose || cfg.Logging.Level == "debug" {
		logLevel := builtin.LogLevelInfo
		if cfg.Logging.Level == "debug" {
			logLevel = builtin.LogLevelDebug
		}

		logFormat := builtin.LogFormatText
		if cfg.Logging.Format == "json" {
			logFormat = builtin.LogFormatJSON
		}

		loggingHook := builtin.NewLoggingHook(
			builtin.WithLogLevel(logLevel),
			builtin.WithLogFormat(logFormat),
		)
		dispatcher.Register(loggingHook)
	}

	// Add audit hook if enabled
	if cfg.Hooks.Audit.Enabled {
		auditOpts := []builtin.AuditOption{}
		if cfg.Hooks.Audit.LogFile != "" {
			auditOpts = append(auditOpts, builtin.WithAuditFile(cfg.Hooks.Audit.LogFile))
		}

		auditHook := builtin.NewAuditHook(true, auditOpts...)
		dispatcher.Register(auditHook)
	}

	return dispatcher
}

// cleanupDispatcher closes any resources held by hooks.
func cleanupDispatcher(dispatcher *hooks.Dispatcher) {
	for _, hook := range dispatcher.Hooks() {
		// Close audit hook if present
		if auditHook, ok := hook.(*builtin.AuditHook); ok {
			_ = auditHook.Close()
		}
	}
}

// =============================================================================
// Configuration
// =============================================================================

// loadConfig loads the application configuration.
func loadConfig() (*config.Config, error) {
	loader := config.NewLoader()

	// Load configuration from file or defaults
	cfg, err := loader.Load(configFile)
	if err != nil {
		// Return default config if no config file found
		return defaultConfig(), nil
	}

	return cfg, nil
}

// defaultConfig returns a default configuration.
func defaultConfig() *config.Config {
	return &config.Config{
		AWS: config.AWSConfig{
			Region: "us-east-1",
		},
		TUI: config.TUIConfig{
			RefreshInterval: 30000000000, // 30s in nanoseconds
			Theme:           "default",
			MouseEnabled:    true,
			AltScreen:       true,
		},
		Services: config.ServicesConfig{
			Enabled: []string{"ec2", "iam", "s3", "lambda"},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		Hooks: config.HooksConfig{
			Audit: config.AuditHookConfig{
				Enabled: false,
			},
		},
	}
}

// applyFlagOverrides applies CLI flags to configuration.
func applyFlagOverrides(cfg *config.Config) {
	if awsProfile != "" {
		cfg.AWS.Profile = awsProfile
	}
	if awsRegion != "" {
		cfg.AWS.Region = awsRegion
	}
	if verbose {
		cfg.Logging.Level = "debug"
	}
}

// =============================================================================
// Service Registration
// =============================================================================

// registerServices registers all enabled services.
func registerServices(reg *registry.Registry, factory *awsfactory.ClientFactory, cfg *config.Config, dispatcher core.EventDispatcher) error {
	// Determine enabled services
	enabledServices := cfg.Services.Enabled
	if len(enabledServices) == 0 {
		enabledServices = []string{"ec2", "iam", "s3", "lambda"}
	}

	// Service registration map
	registrations := map[string]func() (core.ServiceRegistration, error){
		"ec2": func() (core.ServiceRegistration, error) {
			return core.ServiceRegistration{
				Service:     ec2.NewService(factory, dispatcher),
				ViewFactory: ec2.NewViewFactory(),
				Priority:    100,
			}, nil
		},
		"iam": func() (core.ServiceRegistration, error) {
			return core.ServiceRegistration{
				Service:     iam.NewService(factory, dispatcher),
				ViewFactory: iam.NewViewFactory(),
				Priority:    90,
			}, nil
		},
		"s3": func() (core.ServiceRegistration, error) {
			return core.ServiceRegistration{
				Service:     s3.NewService(factory, dispatcher),
				ViewFactory: s3.NewViewFactory(),
				Priority:    80,
			}, nil
		},
		"lambda": func() (core.ServiceRegistration, error) {
			return core.ServiceRegistration{
				Service:     lambda.NewService(factory, dispatcher),
				ViewFactory: lambda.NewViewFactory(),
				Priority:    70,
			}, nil
		},
	}

	// Register enabled services
	for _, name := range enabledServices {
		createFn, ok := registrations[name]
		if !ok {
			continue // Skip unknown services
		}

		registration, err := createFn()
		if err != nil {
			return fmt.Errorf("failed to create %s service: %w", name, err)
		}

		if err := reg.RegisterServiceAndView(registration); err != nil {
			return fmt.Errorf("failed to register %s: %w", name, err)
		}
	}

	return nil
}

// =============================================================================
// CLI Initialization
// =============================================================================

func init() {
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (json|table)")
	rootCmd.PersistentFlags().StringVar(&awsProfile, "profile", "", "AWS profile to use")
	rootCmd.PersistentFlags().StringVar(&awsRegion, "region", "", "AWS region")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Simulate actions without making changes")
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Config file path (optional)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
}
