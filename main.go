package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/modfin/bellman/models/embed"
	"github.com/modfin/blot/internal/ai"
	"github.com/modfin/blot/internal/db"
	"github.com/modfin/clix"
	"github.com/urfave/cli/v3"
	"io"
	"log/slog"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"strings"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {

	defer func() {
		db.Statistics()
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
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}

			return ctx, nil
		},

		Commands: []*cli.Command{
			{
				Name:      "add",
				Usage:     "add to the knowledge base",
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

						dirty, err := queries.Dirty(ctx, label, name, content)
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

						jsonFloats, err := json.Marshal(resp.AsFloat64())
						if err != nil {
							return fmt.Errorf("failed to marshal embedding: %w", err)
						}
						embeddingVector := string(jsonFloats)

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

						logger.Info("Added fragment", "id", frag.ID)
					}

					return nil
				},
			},

			{
				Name:  "search",
				Usage: "search the knowledge base for documents",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "limit",
						Usage:   "the maximum number of documents to return",
						Value:   5,
						Sources: cli.EnvVars("BLOT_LIMIT"),
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
					}

					search := strings.Join(cmd.Args().Slice(), " ")

					resp, err := proxy.Embed(embed.Request{
						Ctx:   ctx,
						Model: model,
						Text:  search,
					})

					data, err := json.Marshal(resp.AsFloat64())
					vec := string(data)
					if err != nil {
						return fmt.Errorf("failed to marshal embedding: %w", err)
					}

					frags, err := queries.KNN(ctx, vec, cmd.Int("limit"), 0)
					if err != nil {
						return fmt.Errorf("failed to query database: %w", err)
					}

					for _, frag := range frags {
						fmt.Printf("============ %s: %s ============\n%s\n", frag.Label, frag.Name, frag.Content)
					}

					return nil
				},
			},

			{
				Name:  "query",
				Usage: "ask a question",
				Flags: []cli.Flag{},
				Action: func(context.Context, *cli.Command) error {
					fmt.Println("Adding a new note")
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Default().Error("got error running blot", "err", err)
	}
}
