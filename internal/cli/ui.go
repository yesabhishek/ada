package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/yesabhishek/ada/internal/ui"
)

func newUICommand() *cobra.Command {
	var (
		host          string
		port          int
		openInBrowser bool
	)
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Start a local dashboard to watch Ada activity.",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFactory(context.Background())
			if err != nil {
				return err
			}
			defer f.Close()

			address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
			server := &http.Server{
				Addr:              address,
				Handler:           ui.New(f.workspace, f.store, f.diff, currentAuthor()).Handler(),
				ReadHeaderTimeout: 5 * time.Second,
			}
			url := "http://" + address
			fmt.Fprintf(cmd.OutOrStdout(), "Ada UI running at %s\n", url)
			fmt.Fprintln(cmd.OutOrStdout(), "Press Ctrl+C to stop.")
			if openInBrowser {
				go openBrowser(url)
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			errCh := make(chan error, 1)
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					errCh <- err
				}
				close(errCh)
			}()

			select {
			case err := <-errCh:
				if err != nil {
					return err
				}
				return nil
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				return server.Shutdown(shutdownCtx)
			}
		},
	}
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "host interface to bind")
	cmd.Flags().IntVar(&port, "port", 4173, "port to bind")
	cmd.Flags().BoolVar(&openInBrowser, "open", false, "open the dashboard in your browser")
	return cmd
}

func openBrowser(url string) {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	_ = command.Start()
}
