package cli

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/Meha555/go-pipeline/internal"
	"github.com/Meha555/go-pipeline/notify/email"
	"github.com/Meha555/go-pipeline/parser"
	"github.com/Meha555/go-pipeline/pipeline"

	"github.com/spf13/cobra"
)

// runCmd 执行pipeline
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a pipeline",
	Long:  "Run a pipeline through a config file",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var conf *parser.PipelineConf
		// 没有被注册到cobra的参数会被认为是额外参数出现在这里的args中
		conf, err = parser.ParseConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("parsing %s failed: %w", configFile, err)
		}

		pipe := pipeline.MakePipeline(conf)

		ctx := context.Background()
		// 处理额外的参数
		parser.ParseArgs(args, ctx)
		if verbose {
			ctx = context.WithValue(ctx, internal.VerboseKey, verbose)
		}
		if noSilence {
			ctx = context.WithValue(ctx, internal.NoSilenceKey, noSilence)
		}
		if trace && !dryRun {
			ctx = context.WithValue(ctx, internal.TraceKey, trace)
		}
		if dryRun {
			ctx = context.WithValue(ctx, internal.DryRunKey, dryRun)
		}

		status := pipe.Run(ctx)

		// Notify 结果
		var eNotifier *email.Sender
		var ebuilder *email.Builder
		if conf.Notifiers != nil {
			if conf.Notifiers.Email != nil {
				eNotifier = &email.Sender{
					SmtpServer: conf.Notifiers.Email.Server,
					SmtpPort:   conf.Notifiers.Email.Port,
					Password:   conf.Notifiers.Email.From.Password,
				}
				toAddrs, err := mail.ParseAddressList(strings.Join(conf.Notifiers.Email.To, ","))
				if err != nil {
					return fmt.Errorf("parsing to addresses failed: %w", err)
				}
				ccAddrs, err := mail.ParseAddressList(strings.Join(conf.Notifiers.Email.Cc, ","))
				if err != nil {
					return fmt.Errorf("parsing cc addresses failed: %w", err)
				}
				ebuilder = email.NewBuilder().
					From(&mail.Address{Name: "go-pipeline", Address: conf.Notifiers.Email.From.Address}).
					To(toAddrs).
					Cc(ccAddrs)
			}
		}

		if status == pipeline.Failed {
			err = fmt.Errorf("pipeline %s@%s run failed", pipe.Name, pipe.Version)
			if conf.Notifiers != nil {
				if conf.Notifiers.Email != nil {
					err = eNotifier.Send(ebuilder.
						Subject("Pipeline Failed").
						Body([]byte(err.Error())).
						Build())
				}
			}
		} else {
			if conf.Notifiers != nil {
				if conf.Notifiers.Email != nil {
					err = eNotifier.Send(ebuilder.
						Subject("Pipeline Success").
						Body([]byte(fmt.Sprintf("pipeline %s@%s run success", pipe.Name, pipe.Version))).
						Build())
				}
			}
		}
		return
	},
}

var (
	configFile string
	verbose    bool
	noSilence  bool
	trace      bool
	dryRun     bool
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().BoolVarP(&noSilence, "no-silence", "s", false, "print every action")
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output for jobs")
	runCmd.Flags().BoolVarP(&trace, "trace", "t", false, "time trace for jobs")
	runCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "dry run")
	runCmd.Flags().StringVarP(&configFile, "file", "f", "", "config file")
	runCmd.MarkFlagRequired("file")
}
