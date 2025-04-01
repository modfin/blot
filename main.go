package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/MatusOllah/slogcolor"
	"github.com/disintegrator/inv"
	"github.com/modfin/bellman/models/embed"
	"github.com/modfin/bellman/models/gen"
	"github.com/modfin/bellman/prompt"
	"github.com/modfin/bellman/schema"
	"github.com/modfin/blot/internal/ai"
	"github.com/modfin/blot/internal/db"
	"github.com/modfin/blot/internal/db/vec"
	"github.com/modfin/clix"
	"github.com/modfin/henry/slicez"
	"github.com/urfave/cli/v3"
	"io"
	"log/slog"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
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
		Name:  "blot",
		Usage: "a RAG LLM tool to creat a knowledge base from file and then answer questions",
		Action: func(context.Context, *cli.Command) error {
			fmt.Println("Nothing to do here yet")
			return nil
		},

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
				Usage:     "takes a row based file and explodes it into one per row in the file",
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

						base := filepath.Base(file)
						out := base + ".exploded"
						if dir != "" {
							out = dir
						}
						dir := filepath.Join(filepath.Dir(file), out)

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

					config := clix.ParseCommand[ai.Config](cmd)
					proxy, err := ai.New(config, slog.Default())
					if err != nil {
						return fmt.Errorf("failed to create proxy: %w", err)
					}

					conn, err := sql.Open("sqlite", cmd.String("db"))
					if err != nil {
						return fmt.Errorf("failed to open database file, %s: %w", "file://"+cmd.String("db"), err)
					}

					_, err = conn.ExecContext(ctx, db.Schema)
					if err != nil {
						return fmt.Errorf("failed to create schema: %w", err)
					}
					queries := db.New(conn)

					embeddingModel := cmd.String("embed-model")
					provider, modelName, _ := strings.Cut(embeddingModel, "/")
					slog.Default().Debug("embedding", "provider", provider, "model", modelName)
					model := embed.Model{
						Provider: provider,
						Name:     modelName,
						Type:     embed.TypeDocument,
					}

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

						dirty, err := queries.DirtyFragment(ctx, label, name, content)
						if err != nil {
							return fmt.Errorf("failed to check if fragment is dirty: %w", err)
						}
						if !dirty {
							logger.Debug("skipping already existing fragment")
							continue
						}

						logger.With("name", name, "label", label)
						logger.Debug("embedding file", "len", len(content))
						resp, err := proxy.Embed(embed.Request{
							Ctx:   ctx,
							Model: model,
							Text:  content,
						})

						if err != nil {
							return fmt.Errorf("failed to embed: %w", err)
						}
						embeddingVector := resp.AsFloat64()

						logger.Debug("adding embedding to database")
						frag, err := queries.AddFragment(context.Background(),
							label,
							name,
							content,
							embeddingModel,
							embeddingVector,
						)

						if err != nil {
							return fmt.Errorf("failed to add fragment: %w", err)
						}

						inv.Require("resulting embedding must mach original",
							"vecors shall be equal", reflect.DeepEqual(embeddingVector, frag.EmbeddingVector))

						logger.Info("Added fragment", "id", frag.ID)
					}

					return nil
				},
			},

			{
				Name:  "search",
				Usage: "search the knowledge base for documents",
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

					query := strings.Join(cmd.Args().Slice(), " ")
					fragments, err := search(ctx, cmd, query)

					if err != nil {
						return fmt.Errorf("failed to search: %w", err)
					}
					for _, frag := range fragments {
						slog.Default().Debug("Found fragment", "id", frag.ID, "label", frag.Label, "name", frag.Name)
						fmt.Println(frag.Name)
						if cmd.Bool("emit") {
							fmt.Println(frag.Content)
							fmt.Println()
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

					query := strings.Join(cmd.Args().Slice(), " ")
					fragments, err := search(ctx, cmd, query)

					if err != nil {
						return fmt.Errorf("failed to search: %w", err)
					}

					config := clix.ParseCommand[ai.Config](cmd)
					proxy, err := ai.New(config, slog.Default())
					if err != nil {
						return fmt.Errorf("failed to create proxy: %w", err)
					}

					llmModel := cmd.String("llm-model")
					if llmModel == "" {
						return fmt.Errorf("llm-model is not set, use --llm-model")
					}

					provider, modelName, _ := strings.Cut(llmModel, "/")
					slog.Default().Debug("llm", "provider", provider, "model", modelName)
					model := gen.Model{
						Provider: provider,
						Name:     modelName,
					}
					llm, err := proxy.Gen(model)
					if err != nil {
						return fmt.Errorf("failed to create llm: %w", err)
					}

					prompts := slicez.Map(fragments, func(frag db.Fragment) prompt.Prompt {
						return prompt.Prompt{
							Role: prompt.UserRole,
							Text: fmt.Sprintf("<%s-document> %s </%s-document>", frag.Label, frag.Content, frag.Label),
						}
					})

					res, err := llm.Model(model).
						System(cmd.String("system-prompt")).
						Output(schema.From(ai.Answer{})).
						Prompt(append(prompts, prompt.Prompt{
							Role: prompt.UserRole,
							Text: fmt.Sprintf("<user-question> %s </user-question>", query),
						})...)

					if err != nil {
						return fmt.Errorf("failed to generate response: %w", err)
					}

					var ans ai.Answer
					err = res.Unmarshal(&ans)
					if err != nil {
						return fmt.Errorf("failed to unmarshal response: %w", err)
					}

					slog.Default().Info("llm statistics",
						"tokens-input", res.Metadata.InputTokens,
						"tokens-output", res.Metadata.OutputTokens,
						"tokens-total", res.Metadata.TotalTokens,
						"model", res.Metadata.Model,
						"confidence", ans.ConfidenceScore,
					)

					fmt.Println(ans.Answer)

					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Default().Error("got error running blot", "err", err)
	}
}

func search(ctx context.Context, cmd *cli.Command, query string) ([]db.Fragment, error) {
	config := clix.ParseCommand[ai.Config](cmd)
	proxy, err := ai.New(config, slog.Default())
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy: %w", err)
	}

	conn, err := sql.Open("sqlite", cmd.String("db"))
	if err != nil {
		return nil, fmt.Errorf("failed to open database file, %s: %w", "file://"+cmd.String("db"), err)
	}

	_, err = conn.ExecContext(ctx, db.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}
	queries := db.New(conn)

	embeddingModel := cmd.String("embed-model")
	provider, modelName, _ := strings.Cut(embeddingModel, "/")
	slog.Default().Debug("embedding", "provider", provider, "model", modelName)
	model := embed.Model{
		Provider: provider,
		Name:     modelName,
		Type:     embed.TypeQuery,
	}

	resp, err := proxy.Embed(embed.Request{
		Ctx:   ctx,
		Model: model,
		Text:  query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to embed: %w", err)
	}

	vector := resp.AsFloat64()

	var fragments []db.Fragment
	for _, lim := range cmd.StringSlice("limit") {
		label, strlimit, found := strings.Cut(lim, ":")
		if !found {
			strlimit = label
			label = "%"
		}

		limit, err := strconv.Atoi(strlimit)
		if err != nil {
			slog.Default().Warn("failed to parse limit, defaulting to 5", "err", err)
			limit = 5
		}

		slog.Default().Debug("searching for fragments", "label", label, "limit", limit)

		frags, err := queries.KNN(ctx, vector, label, int64(limit))
		if err != nil {
			return nil, fmt.Errorf("failed knn to query database : %w", err)
		}

		fragments = append(fragments, frags...)
	}

	return slicez.UniqBy(fragments, func(a db.Fragment) int64 {
		return a.ID
	}), nil

}
