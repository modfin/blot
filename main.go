package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/MatusOllah/slogcolor"
	"github.com/disintegrator/inv"
	"github.com/modfin/bellman/models/embed"
	"github.com/modfin/blot/internal/ai"
	"github.com/modfin/blot/internal/db/vec"
	"github.com/urfave/cli/v3"
	"io"
	"log/slog"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {

	options := slogcolor.DefaultOptions
	handler := slogcolor.NewHandler(os.Stderr, options)
	slog.SetDefault(slog.New(handler))

	defer func() {
		vec.Statistics()
	}()

	cmd := &cli.Command{
		Name:      "blot",
		UsageText: "blot [global options] command [command options] [arguments...]",
		Usage: "with a rag, tool to creat a knowledge base from files and\n" +
			"answer questions about it",
		Description: `Bolt is a cli rag llm tool that can be used to create a knowledge base from files.
With the knowledge base, you can search, ask questions and fill / autocomplete a csv file.

blot is based around Bellman https://github.com/modfin/bellman which enables the user to pick an choose what
llm and embedding models to use from implemented vendors, ie. OpenAI, VertexAI, Anthropic and VoyageAI or a Bellman proxy.
`,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "db",
				Value:   "./blot.db",
				Usage:   "Path to database file",
				Sources: cli.EnvVars("BLOT_DB"),
			},

			&cli.StringFlag{
				Name:    "bellman-url",
				Sources: cli.EnvVars("BLOT_BELLMAN_URL"),
			},
			&cli.StringFlag{
				Name:    "bellman-key",
				Sources: cli.EnvVars("BLOT_BELLMAN_KEY"),
			},
			&cli.StringFlag{
				Name:    "bellman-key-name",
				Value:   "blot",
				Sources: cli.EnvVars("BLOT_BELLMAN_KEY_NAME"),
			},

			&cli.StringFlag{
				Name:    "vertexai-credential",
				Sources: cli.EnvVars("BLOT_VERTEXAI_CREDENTIAL"),
			},
			&cli.StringFlag{
				Name:    "vertexai-project",
				Sources: cli.EnvVars("BLOT_VERTEXAI_PROJECT"),
			},
			&cli.StringFlag{
				Name:    "vertexai-region",
				Sources: cli.EnvVars("BLOT_VERTEXAI_REGION"),
			},

			&cli.StringFlag{
				Name:    "openai-key",
				Sources: cli.EnvVars("BLOT_OPENAI_KEY"),
			},
			&cli.StringFlag{
				Name:    "anthropic-key",
				Sources: cli.EnvVars("BLOT_ANTHROPIC_KEY"),
			},
			&cli.StringFlag{
				Name:    "voyageai-key",
				Sources: cli.EnvVars("BLOT_VOYAGEAI_KEY"),
			},

			&cli.StringFlag{
				Name:    "embed-model",
				Value:   "OpenAI/text-embedding-3-small",
				Sources: cli.EnvVars("BLOT_EMBED_MODEL"),
			},
			&cli.StringFlag{
				Name:    "llm-model",
				Value:   "OpenAI/gpt-4o-mini",
				Sources: cli.EnvVars("BLOT_LLM_MODEL"),
			},

			&cli.BoolFlag{
				Name:    "verbose",
				Sources: cli.EnvVars("BLOT_VERBOSE"),
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {

			if cmd.Bool("verbose") {
				options.Level = slog.LevelDebug
				handler = slogcolor.NewHandler(os.Stderr, options)
				slog.SetDefault(slog.New(handler))
			}

			return ctx, nil
		},

		Commands: []*cli.Command{
			{
				Name:      "explode",
				Usage:     "takes a row based file and explodes it into one file per row in the file",
				ArgsUsage: "<file>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "out",
						Usage:   "dir in witch to put the resulting files",
						Sources: cli.EnvVars("BLOT_OUT"),
					},
					&cli.StringFlag{
						Name:    "delimiter",
						Aliases: []string{"d"},
						Value:   "\\t",
						Usage:   "delimiter for separating columns",
						Sources: cli.EnvVars("BLOT_DELIMITER"),
					},
					&cli.BoolFlag{
						Name:    "with-headers",
						Sources: cli.EnvVars("BLOT_WITH_HEADERS"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {

					dir := cmd.String("out")

					for _, file := range cmd.Args().Slice() {
						dir := dir

						base := filepath.Base(file)
						if dir == "" {
							dir = filepath.Join(filepath.Dir(file), base+".exploded")
						}

						err := os.MkdirAll(dir, 0755)
						if err != nil {
							return err
						}

						delimiter := cmd.String("delimiter")

						in, err := os.Open(file)
						if err != nil {
							return err
						}
						r := csv.NewReader(in)
						r.LazyQuotes = true
						switch delimiter {
						case "\\t":
							r.Comma = '\t'
						default:
							r.Comma = rune(delimiter[0])
						}

						var headers []string
						getName := func(col int) string {
							if len(headers) > col {
								return headers[col]
							}
							return fmt.Sprintf("col_%d", col)
						}

						writeRow := func(i int, record []string) error {

							outfile := filepath.Join(dir, fmt.Sprintf("%04d_%s", i, base))
							slog.Default().Debug("writing", "file", outfile)

							var buf strings.Builder
							for j, col := range record {
								buf.WriteString(getName(j))
								buf.WriteString(":\t")
								buf.WriteString(col)
								buf.WriteString("\n")
							}
							return os.WriteFile(outfile, []byte(buf.String()), 0644)
						}

						var row int
						for {

							row++
							record, err := r.Read()
							if err == io.EOF {
								break
							}
							if row == 1 && cmd.Bool("with-headers") {
								headers = append([]string{}, record...)
								continue
							}

							err = writeRow(row-1, record)
							if err != nil {
								return fmt.Errorf("failed to write row %d: %w", row-1, err)
							}
						}

					}

					return nil
				},
			},
			{

				Name:      "add",
				Usage:     "adds a file to the knowledge base",
				ArgsUsage: "<file>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "label",
						Usage:   "the label for the note",
						Value:   "default",
						Sources: cli.EnvVars("BLOT_LABEL"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {

					cfg, err := ai.LoadConf(ctx, cmd)
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}

					model := cfg.EmbedModel
					model.Type = embed.TypeDocument

					for _, f := range cmd.Args().Slice() {
						logger := slog.Default().With("file", f)

						logger.Debug("reading file")

						in, err := os.Open(f)
						if err != nil {
							return fmt.Errorf("failed to open file %s: %w", f, err)
						}

						data, err := io.ReadAll(in)
						if err != nil {
							return fmt.Errorf("failed to read file %s: %w", f, err)
						}
						in.Close()

						label := cmd.String("label")
						name := filepath.Clean(f)
						content := string(data)

						dirty, err := cfg.Dao.DirtyFragment(ctx, label, name, content)
						if err != nil {
							return fmt.Errorf("failed to check if fragment is dirty: %w", err)
						}
						if !dirty {
							logger.Debug("skipping already existing fragment")
							continue
						}

						logger = logger.With("name", name, "label", label)
						logger.Debug("embedding file", "len", len(content))
						resp, err := cfg.Proxy.Embed(embed.Request{
							Ctx:   ctx,
							Model: model,
							Text:  content,
						})

						if err != nil {
							return fmt.Errorf("failed to embed: %w", err)
						}
						embeddingVector := resp.AsFloat64()

						logger.Debug("adding embedding to database")
						frag, err := cfg.Dao.AddFragment(context.Background(),
							label,
							name,
							content,
							model.String(),
							embeddingVector,
						)

						if err != nil {
							return fmt.Errorf("failed to add fragment: %w", err)
						}

						inv.Require("resulting embedding must mach original",
							"vectors shall be equal", reflect.DeepEqual(embeddingVector, frag.EmbeddingVector))

						logger.Info("Added fragment", "id", frag.ID)
					}

					return nil
				},
			},

			{
				Name:  "search",
				Usage: "Search the knowledge base for documents",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name: "emit",
					},
					&cli.StringSliceFlag{
						Name: "limit",
						Usage: "the maximum number of documents to return. \n" +
							"eg. --limit=5, but can be further broken down by label.\n" +
							"--limit=QA:3 --limit=policies:2 --limit=procedures:1. \n" +
							"Resulting in 6 fragments returned ",
						Value:   []string{"5"},
						Sources: cli.EnvVars("BLOT_LIMIT"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {

					cfg, err := ai.LoadConf(ctx, cmd)
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}

					question := strings.Join(cmd.Args().Slice(), " ")
					fragments, err := ai.Search(cfg, question)

					if err != nil {
						return fmt.Errorf("failed to Search: %w", err)
					}
					for _, frag := range fragments {
						slog.Default().Debug("Found fragment", "id", frag.ID, "label", frag.Label, "name", frag.Name)
						fmt.Println(frag.Name)
						if cmd.Bool("emit") {
							fmt.Println(frag.Content)
						}
					}

					return nil
				},
			},

			{
				Name:  "prompt",
				Usage: "ask a question about the knowledge base",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "system-prompt",
						Usage: "the system prompt to use that will be used for the prompt when RAGing.",
					},
					&cli.StringSliceFlag{
						Name: "limit",
						Usage: "the maximum number of documents to that is used for the prompt when RAGing. \n" +
							"eg. --limit=5, but can be further broken down by label.\n" +
							"--limit=QA:3 --limit=policies:2 --limit=procedures:1. \n" +
							"Resulting in 6 fragments returned ",
						Value:   []string{"5"},
						Sources: cli.EnvVars("BLOT_LIMIT"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {

					cfg, err := ai.LoadConf(ctx, cmd)
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}
					question := strings.Join(cmd.Args().Slice(), " ")

					ans, err := ai.Query(cfg, question)
					if err != nil {
						return fmt.Errorf("failed to Query: %w", err)
					}

					slog.Default().Debug("llm statistics",
						"tokens-input", ans.Metadata.InputTokens,
						"tokens-output", ans.Metadata.OutputTokens,
						"tokens-total", ans.Metadata.TotalTokens,
						"model", ans.Metadata.Model,
						"confidence", ans.ConfidenceScore,
					)

					fmt.Println(ans.Answer)

					return nil
				},
			},
			{
				Name: "fill",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "in",
						Usage:   "the input file",
						Sources: cli.EnvVars("BLOT_IN"),
					},
					&cli.StringFlag{
						Name:    "out",
						Usage:   "the output file",
						Sources: cli.EnvVars("BLOT_OUT"),
					},
					&cli.StringFlag{
						Name:    "delimiter",
						Aliases: []string{"d"},
						Value:   "\\t",
						Usage:   "delimiter for separating columns",
						Sources: cli.EnvVars("BLOT_DELIMITER"),
					},
					&cli.BoolFlag{
						Name:    "with-headers",
						Sources: cli.EnvVars("BLOT_WITH_HEADERS"),
					},
					&cli.StringFlag{
						Name:    "system-prompt",
						Usage:   "the system prompt to use that will be used for the prompt when RAGing.",
						Sources: cli.EnvVars("BLOT_SYSTEM_PROMPT"),
					},
					&cli.StringSliceFlag{
						Name: "limit",
						Usage: "the maximum number of documents to that is used for the prompt when RAGing. \n" +
							"eg. --limit=5, but can be further broken down by label.\n" +
							"--limit=QA:3 --limit=policies:2 --limit=procedures:1. \n" +
							"Resulting in 6 fragments returned ",
						Value:   []string{"5"},
						Sources: cli.EnvVars("BLOT_LIMIT"),
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {

					cfg, err := ai.LoadConf(ctx, cmd)
					if err != nil {
						return fmt.Errorf("failed to load config: %w", err)
					}

					return ai.Fill(cfg)

				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Default().Error("got error running blot", "err", err)
	}
}
