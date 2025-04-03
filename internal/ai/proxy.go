package ai

import (
	"errors"
	"fmt"
	"github.com/modfin/bellman"
	"github.com/modfin/bellman/models/embed"
	"github.com/modfin/bellman/models/gen"
	"github.com/modfin/bellman/services/anthropic"
	"github.com/modfin/bellman/services/openai"
	"github.com/modfin/bellman/services/vertexai"
	"github.com/modfin/bellman/services/voyageai"
	"log/slog"
	"strings"
)

type APICredentials struct {
	BellmanURL     string `cli:"bellman-url"`
	BellmanKeyName string `cli:"bellman-key-name"`
	BellmanKey     string `cli:"bellman-key"`

	VertexAICredential string `cli:"vertexai-credential"`
	VertexAIProject    string `cli:"vertexai-project"`
	VertexAIRegion     string `cli:"vertexai-region"`

	OpenAIKey    string `cli:"openai-key"`
	AnthropicKey string `cli:"anthropic-key"`
	VoyageAIKey  string `cli:"voyageai-key"`
}

func New(credentials APICredentials, logger *slog.Logger) (*Proxy, error) {
	proxy := newProxy()

	if credentials.AnthropicKey != "" {
		client := anthropic.New(credentials.AnthropicKey)

		proxy.RegisterGen(client)

		logger.Debug("adding llm provider", "provider", client.Provider())

	}
	if credentials.OpenAIKey != "" {
		client := openai.New(credentials.OpenAIKey)

		proxy.RegisterGen(client)

		logger.Debug("adding llm provider", "provider", client.Provider())

		proxy.RegisterEmbeder(client)
		logger.Debug("adding embed provider", "provider", client.Provider())

	}

	if credentials.VertexAIRegion != "" && credentials.VertexAIProject != "" {
		var err error
		client, err := vertexai.New(vertexai.GoogleConfig{
			Project:    credentials.VertexAIProject,
			Region:     credentials.VertexAIRegion,
			Credential: credentials.VertexAICredential,
		})
		if err != nil {
			return nil, err
		}

		proxy.RegisterGen(client)
		logger.Debug("adding llm provider", "provider", client.Provider())

		proxy.RegisterEmbeder(client)
		logger.Debug("adding embed provider", "provider", client.Provider())

	}

	if credentials.VoyageAIKey != "" {
		client := voyageai.New(credentials.VoyageAIKey)
		proxy.RegisterEmbeder(client)
		logger.Debug("adding embed provider", "provider", client.Provider())
	}

	if credentials.BellmanKey != "" && credentials.BellmanURL != "" {
		client := bellman.New(credentials.BellmanURL, bellman.Key{
			Name:  credentials.BellmanKeyName,
			Token: credentials.BellmanKey,
		})
		proxy.RegisterGen(client)
		logger.Debug("adding llm provider", "provider", client.Provider())

		proxy.RegisterEmbeder(client)
		logger.Debug("adding embed provider", "provider", client.Provider())
	}

	return proxy, nil

}

var ErrNoModelProvided = errors.New("no model was provided")
var ErrClientNotFound = errors.New("client not found")

type Proxy struct {
	embeders map[string]embed.Embeder
	gens     map[string]gen.Gen
}

func newProxy() *Proxy {
	p := &Proxy{
		embeders: map[string]embed.Embeder{},
		gens:     map[string]gen.Gen{},
	}

	return p
}

func (p *Proxy) RegisterEmbeder(embeder embed.Embeder) {
	p.embeders[embeder.Provider()] = embeder
}
func (p *Proxy) RegisterGen(llm gen.Gen) {
	p.gens[llm.Provider()] = llm
}

func (p *Proxy) Embed(mod embed.Request) (*embed.Response, error) {
	client, ok := p.embeders[mod.Model.Provider]
	if !ok {
		return nil, fmt.Errorf("no client registerd for provider '%s', %w", mod.Model.Provider, ErrClientNotFound)
	}

	if client == nil {
		return nil, ErrNoModelProvided
	}

	if mod.Model.Provider == bellman.Provider {
		provider, name, found := strings.Cut(mod.Model.Name, "/")

		if !found {
			return nil, fmt.Errorf("invalid bellman model name '%s', %w", mod.Model.Name, ErrNoModelProvided)
		}
		mod.Model.Provider = provider
		mod.Model.Name = name

	}

	if mod.Model.Name == "" {
		return nil, fmt.Errorf("mod.Model.Name is not set, %w", ErrNoModelProvided)
	}
	return client.Embed(mod)
}

func (p *Proxy) Gen(mod gen.Model) (*gen.Generator, error) {
	client, ok := p.gens[mod.Provider]
	if !ok {
		return nil, fmt.Errorf("no client registerd for provider '%s', %w", mod.Provider, ErrClientNotFound)
	}

	if client == nil {
		return nil, ErrClientNotFound
	}

	if mod.Provider == bellman.Provider {
		provider, name, found := strings.Cut(mod.Name, "/")

		if !found {
			return nil, fmt.Errorf("invalid bellman model name '%s', %w", mod.Name, ErrNoModelProvided)
		}
		mod.Provider = provider
		mod.Name = name
	}

	if mod.Name == "" {
		return nil, fmt.Errorf("mod.Name is not set, %w", ErrNoModelProvided)
	}

	return client.Generator(gen.WithModel(mod)), nil
}
