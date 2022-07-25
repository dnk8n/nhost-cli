/*
MIT License

Copyright (c) Nhost

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/nhost/cli/logger"
	"github.com/nhost/cli/nhost/compose"
	"github.com/nhost/cli/nhost/service"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/nhost/cli/nhost"
	"github.com/nhost/cli/util"
	"github.com/spf13/cobra"
)

var (
	//  initialize boolean to determine
	//  whether to expose the local environment to the public internet
	//  through a tunnel
	expose    bool
	proxyPort string
	//  signal interruption channel
	stop = make(chan os.Signal, 1)
)

//  devCmd represents the dev command
var devCmd = &cobra.Command{
	Use:        "dev [-p port]",
	Aliases:    []string{"up"},
	SuggestFor: []string{"list", "init"},
	Short:      "Start local development environment",
	Long:       `Initialize a local Nhost environment for development and testing.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {

		//  check if nhost/ exists
		if !util.PathExists(nhost.NHOST_DIR) {
			status.Infoln("Initialize new app by running 'nhost init'")
			return errors.New("app not found in this directory")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var mgr service.Manager

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		config, err := nhost.GetConfiguration()
		if err != nil {
			return err
		}

		projectName, err := nhost.GetDockerComposeProjectName()
		if err != nil {
			return err
		}

		env, err := nhost.Env()
		if err != nil {
			return fmt.Errorf("failed to read .env.development: %v", err)
		}

		proxyPort, err := strconv.Atoi(cmd.Flag("port").Value.String())
		if err != nil {
			return fmt.Errorf("failed to parse port: %v", err)
		}

		ports := compose.NewPorts(uint32(proxyPort))
		mgr = service.NewDockerComposeManager(config, ports, env, nhost.GetCurrentBranch(), projectName, log, status, logger.DEBUG)

		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			err = mgr.SyncExec(ctx, func(ctx context.Context) error {
				startCtx, cancel := context.WithTimeout(ctx, time.Minute*3)
				defer cancel()

				return retry.Do(func() error {
					return mgr.Start(startCtx)
				}, retry.Attempts(3))
			})

			if ctx.Err() == context.Canceled {
				return
			}

			if err != nil {
				status.Errorln("Failed to start services")
				log.WithError(err).Error("Failed to start services")
				os.Exit(1)
			}

			if !noBrowser {
				openbrowser(fmt.Sprintf("http://localhost:%d", ports[compose.SvcTraefik]))
			}
		}()

		// wait for stop signal
		<-stop
		cancel()

		exitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		status.Executing("Exiting...")
		log.Debug("Exiting...")
		err = mgr.SyncExec(exitCtx, func(ctx context.Context) error {
			return mgr.Stop(exitCtx)
		})
		if err != nil {
			status.Errorln("Failed to stop services")
		}

		return nil
	},
}

type Printer struct {
	*tabwriter.Writer
}

func newPrinter() *Printer {
	t := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
	return &Printer{
		Writer: t,
	}
}

func (p *Printer) print(loc, head, tail string) {

	switch loc {
	case "header":
		fmt.Fprintln(p)
		//	fmt.Fprintln(p, "---\t\t-----")
	case "footer":
		//	fmt.Fprintln(p, "---\t\t-----")
	case "info":
		status.Info(head)
	default:
		fmt.Fprintf(p, "%v\t\t%v", head, tail)
		fmt.Fprintln(p)
	}
}

func (p *Printer) close() {
	p.Flush()
	fmt.Println()
}

func init() {
	rootCmd.AddCommand(devCmd)

	//  Here you will define your flags and configuration settings.

	//  Cobra supports Persistent Flags which will work for this command
	//  and all subcommands, e.g.:
	devCmd.PersistentFlags().StringVarP(&proxyPort, "port", "p", "1337", "Port for dev proxy")
	devCmd.PersistentFlags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser windows automatically")
}
