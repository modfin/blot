package ai

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/modfin/bellman/models/embed"
	"github.com/modfin/bellman/models/gen"
	"github.com/modfin/bellman/prompt"
	"github.com/modfin/bellman/schema"
	"github.com/modfin/blot/internal/db"
	"github.com/modfin/clix"
	"github.com/modfin/henry/mapz"
	"github.com/modfin/henry/slicez"
	"github.com/urfave/cli/v3"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Conf struct {
	ctx         context.Context
	credentials APICredentials
	Dao         *db.Queries
	Proxy       *Proxy

	EmbedModel embed.Model
	LLMModel   gen.Model

	SystemPrompt string

	limits      map[string]int
	in          string
	out         string
	delimiter   string
	withHeaders bool
}

func LoadConf(ctx context.Context, cmd *cli.Command) (*Conf, error) {
	var err error
	var conf Conf
	conf.ctx = ctx
	conf.credentials = clix.ParseCommand[APICredentials](cmd)
	conf.Proxy, err = New(conf.credentials, slog.Default())
	if err != nil {
		return nil, fmt.Errorf("failed to create Proxy: %w", err)
	}

	conn, err := sql.Open("sqlite", cmd.String("db"))
	if err != nil {
		return nil, fmt.Errorf("failed to open database file, %s: %w", "file://"+cmd.String("db"), err)
	}
	_, err = conn.ExecContext(ctx, db.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}
	conf.Dao = db.New(conn)

	embeddingModel := cmd.String("embed-model")
	provider, modelName, _ := strings.Cut(embeddingModel, "/")

	slog.Default().Debug("embed model", "provider", provider, "model", modelName)

	conf.EmbedModel = embed.Model{
		Provider: provider,
		Name:     modelName,
	}

	llmModel := cmd.String("llm-model")
	provider, modelName, _ = strings.Cut(llmModel, "/")
	slog.Default().Debug("llm model", "provider", provider, "model", modelName)
	conf.LLMModel = gen.Model{
		Provider: provider,
		Name:     modelName,
	}

	conf.SystemPrompt = cmd.String("system-prompt")

	conf.limits = slicez.Associate(cmd.StringSlice("limit"), func(lim string) (key string, value int) {
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
		return label, limit
	})

	conf.in = cmd.String("in")
	conf.out = cmd.String("out")
	conf.delimiter = cmd.String("delimiter")
	conf.withHeaders = cmd.Bool("with-headers")

	return &conf, nil

}

func Fill(cfg *Conf) error {

	in, err := os.Open(cfg.in)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer in.Close()

	out, err := os.Create(cfg.out)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	delimiter := cfg.delimiter
	if len(delimiter) == 0 {
		delimiter = "\t"
	}

	csvin := csv.NewReader(in)
	csvin.LazyQuotes = true
	switch delimiter {
	case "\\t":
		csvin.Comma = '\t'
	default:
		csvin.Comma = rune(delimiter[0])
	}

	csvout := csv.NewWriter(out)
	csvout.Comma = csvin.Comma
	defer csvout.Flush()

	var headers []string
	getName := func(col int) string {
		if len(headers) > col {
			return headers[col]
		}
		return fmt.Sprintf("col_%d", col)
	}

	var inputTokens int
	var outputTokens int

	var row int
	for {
		start := time.Now()

		row++
		record, err := csvin.Read()
		if err == io.EOF {
			break
		}
		if row == 1 && cfg.withHeaders {
			headers = append([]string{}, record...)
			err = csvout.Write(append(record, "answer", "confidence_score"))
			if err != nil {
				return err
			}
			continue
		}

		var col int
		cols := slicez.Map(record, func(s string) string {
			name := getName(col)
			col += 1
			return fmt.Sprintf("<%s>\n  %s\n</%s>", name, s, name)
		})

		answer, err := Query(cfg, strings.Join(cols, "\n"))
		if err != nil {
			return fmt.Errorf("failed to Query: %w", err)
		}

		inputTokens += answer.Metadata.InputTokens
		outputTokens += answer.Metadata.OutputTokens
		err = csvout.Write(append(record, answer.Answer, fmt.Sprintf("%.3f", answer.ConfidenceScore)))
		if err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
		csvout.Flush()

		slog.Default().Debug("Fill",
			"row", row,
			"confidence", answer.ConfidenceScore,
			"took", time.Since(start),
			"input-tokens", answer.Metadata.InputTokens,
			"output-tokens", answer.Metadata.OutputTokens,
			"input-tokens-total", inputTokens,
			"output-tokens-total", outputTokens,
		)
	}

	return nil

}

func Search(cfg *Conf, question string) ([]db.Fragment, error) {

	model := cfg.EmbedModel
	model.Type = embed.TypeQuery

	resp, err := cfg.Proxy.Embed(embed.Request{
		Ctx:   cfg.ctx,
		Model: model,
		Text:  question,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to embed: %w", err)
	}

	vector := resp.AsFloat64()

	fragments := slicez.FlatMap(mapz.Entries(cfg.limits), func(e mapz.Entry[string, int]) []db.Fragment {
		frags, err := cfg.Dao.KNN(cfg.ctx, vector, e.Key, e.Value)
		if err != nil {
			slog.Default().Warn("failed to Query database for fragments", "err", err)
		}

		return frags
	})

	return slicez.UniqBy(fragments, func(a db.Fragment) int {
		return a.ID
	}), nil

}

func Query(cfg *Conf, question string) (Answer, error) {

	fragments, err := Search(cfg, question)
	if err != nil {
		return Answer{}, fmt.Errorf("failed to Search: %w", err)
	}

	llm, err := cfg.Proxy.Gen(cfg.LLMModel)
	if err != nil {
		return Answer{}, fmt.Errorf("failed to create llm: %w", err)
	}

	prompts := slicez.Map(fragments, func(frag db.Fragment) prompt.Prompt {
		return prompt.Prompt{
			Role: prompt.UserRole,
			Text: fmt.Sprintf("<%s-document> %s </%s-document>", frag.Label, frag.Content, frag.Label),
		}
	})

	res, err := llm.
		System(cfg.SystemPrompt).
		Output(schema.From(Answer{})).
		Prompt(append(prompts, prompt.Prompt{
			Role: prompt.UserRole,
			Text: fmt.Sprintf("<user-question> %s </user-question>", question),
		})...)

	if err != nil {
		return Answer{}, fmt.Errorf("failed to generate response: %w", err)
	}

	var ans Answer
	err = res.Unmarshal(&ans)
	if err != nil {
		return Answer{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	ans.Metadata = res.Metadata

	return ans, nil
}
