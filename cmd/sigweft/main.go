// Sigweft generates email signatures from external organisation configuration.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/spf13/cobra"
	"github.com/woodleighschool/sigweft/internal/account"
	"github.com/woodleighschool/sigweft/internal/config"
	"github.com/woodleighschool/sigweft/internal/server"
	"github.com/woodleighschool/sigweft/internal/signature"
	"github.com/woodleighschool/sigweft/web"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	opts, err := config.LoadRuntime()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	root := &cobra.Command{Use: "sigweft", Short: "Generate consistent email signatures", Version: version, SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().StringVar(&opts.Config, "config", opts.Config, "Path to the organisation YAML configuration")
	root.AddCommand(&cobra.Command{Use: "serve", Short: "Serve the signature generator", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			renderer, err := signature.Load(opts.Config)
			if err != nil {
				return err
			}
			auth, err := account.New(cmd.Context(), opts, logger)
			if err != nil {
				return err
			}
			handler, _ := server.New(renderer, web.Files(), logger, version, auth)
			return serve(cmd.Context(), opts.Listen, handler, logger)
		}})
	root.AddCommand(&cobra.Command{Use: "validate", Short: "Validate configuration, template and assets", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := signature.Load(opts.Config); err != nil {
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Configuration is valid.")
			return err
		}})
	root.AddCommand(&cobra.Command{Use: "schema", Short: "Print the configuration JSON Schema", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return writeJSON(cmd.OutOrStdout(), jsonschema.Reflect(&config.Config{}))
		}})
	root.AddCommand(&cobra.Command{Use: "openapi", Short: "Print the API schema", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, api := server.New(nil, nil, logger, version, nil)
			return writeJSON(cmd.OutOrStdout(), api.OpenAPI())
		}})
	return root.ExecuteContext(ctx)
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	return nil
}

func serve(ctx context.Context, listen string, handler http.Handler, logger *slog.Logger) error {
	srv := &http.Server{Addr: listen, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	done := make(chan error, 1)
	go func() { done <- srv.ListenAndServe() }()
	logger.InfoContext(ctx, "serving", "listen", listen, "version", version)
	select {
	case err := <-done:
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("stop HTTP server: %w", err)
		}
		if err := <-done; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
