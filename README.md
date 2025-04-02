# Blot

Blot is a CLI RAG (Retrieval-Augmented Generation) tool that helps you create a knowledge base from files and ask questions about it. With Blot, you can search, ask questions, and fill/autocomplete CSV files using the knowledge you've stored.

## Overview

Blot is built around [Bellman](https://github.com/modfin/bellman), which enables you to choose which LLM and embedding models to use from implemented vendors including OpenAI, VertexAI, Anthropic, and VoyageAI, or a Bellman proxy.

## Installation

```
go install github.com/modfin/blot@latest
```


## Examples 

See examples use case walk through [example/README.md](example/README.md)



## Configuration

Blot can be configured using environment variables or command-line flags:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--db` | `BLOT_DB` | `./blot.db` | Path to database file |
| `--bellman-url` | `BLOT_BELLMAN_URL` | | URL to Bellman service |
| `--bellman-key` | `BLOT_BELLMAN_KEY` | | API key for Bellman |
| `--bellman-key-name` | `BLOT_BELLMAN_KEY_NAME` | `blot` | Key name for Bellman |
| `--vertexai-credential` | `BLOT_VERTEXAI_CREDENTIAL` | | VertexAI credentials |
| `--vertexai-project` | `BLOT_VERTEXAI_PROJECT` | | VertexAI project |
| `--vertexai-region` | `BLOT_VERTEXAI_REGION` | | VertexAI region |
| `--openai-key` | `BLOT_OPENAI_KEY` | | OpenAI API key |
| `--anthropic-key` | `BLOT_ANTHROPIC_KEY` | | Anthropic API key |
| `--voyageai-key` | `BLOT_VOYAGEAI_KEY` | | VoyageAI API key |
| `--embed-model` | `BLOT_EMBED_MODEL` | `OpenAI/text-embedding-3-small` | Embedding model to use |
| `--llm-model` | `BLOT_LLM_MODEL` | `OpenAI/gpt-4o-mini` | LLM model to use |
| `--verbose` | `BLOT_VERBOSE` | `false` | Enable verbose logging |

## Commands

### Explode

Explodes a row-based file, such as csv or tsv, into one file per row.

```
blot [options] explode [options] <file>
```

Options:
- `--out`: Directory in which to put the resulting files (`BLOT_OUT`)
- `--delimiter, -d`: Delimiter for separating columns (default: `\t`) (`BLOT_DELIMITER`)
- `--with-headers`: Use the first row as headers (`BLOT_WITH_HEADERS`)

### Add

Adds a file to the knowledge base.

```
blot [options] add [options] <files ...>
```

Options:
- `--label`: The label for the note (default: `default`) (`BLOT_LABEL`)

### Search

Searches the knowledge base for documents.

```
blot [options] search [options] <query>
```

Options:
- `--emit`: Output the content of found fragments
- `--limit`: Maximum number of documents to return (default: `5`) (`BLOT_LIMIT`)
    - Can be further broken down by label, e.g., `--limit=QA:3 --limit=policies:2`

### Prompt

Asks a question about the knowledge base.

```
blot [options] prompt [options] <question>
```

Options:
- `--system-prompt`: System prompt to use for RAG
- `--limit`: Maximum number of documents to use for the prompt (default: `5`) (`BLOT_LIMIT`)
    - Can be further broken down by label, e.g., `--limit=QA:3 --limit=policies:2`

### Fill

Fills or autocompletes a CSV file using the knowledge base.

```
blot fill [options]
```

Options:
- `--in`: Input file (`BLOT_IN`)
- `--out`: Output file (`BLOT_OUT`)
- `--delimiter, -d`: Delimiter for separating columns (default: `\t`) (`BLOT_DELIMITER`)
- `--with-headers`: Use the first row as headers (`BLOT_WITH_HEADERS`)
- `--system-prompt`: System prompt to use for RAG (`BLOT_SYSTEM_PROMPT`)
- `--limit`: Maximum number of documents to use for the prompt (default: `5`) (`BLOT_LIMIT`)


## LLM and Embedding, provider and models

### LLM Providers
- Anthropic https://docs.anthropic.com/en/docs/about-claude/models/all-models
- OpenAI https://platform.openai.com/docs/models
- VertexAI https://cloud.google.com/vertex-ai/generative-ai/docs/learn/models
- Bellman

### Embedding Providers 
- OpenAI https://platform.openai.com/docs/models#embeddings
- VertexAI  https://cloud.google.com/vertex-ai/generative-ai/docs/learn/models#models
- VoyageAI  https://docs.voyageai.com/docs/embeddings
- Bellman

### Usage
The bellman notation / fqn is used to specify a provider and model.

For example,
- `ChatGPT 4o-mini` would be \
`--llm-model=OpenAI/gpt-4o-mini`
- `Gemini Flash 2.0` would be \
`--llm-model="VertexAI/gemini-2.0-flash-001"`
- `Claude 3.7` would be \
`--llm-model="Anthropic/claude-3-7-sonnet-latest"`
- `text-embedding-3-small` would be \
`--embed-model=OpenAI/text-embedding-3-small`
- `text-embedding-005` would be \
`--embed-model=VertexAI/text-embedding-005`
- `voyage-law-2` would be \
`--embed-model=VoyageAI/voyage-law-2`



### Adding Files to the Knowledge Base

```bash
# Add a single file with the default label
blot --openai-key=$(cat ./files/openai.key) add document.txt

# Add a file with a custom label
blot --openai-key=$(cat ./files/openai.key) add --label=policies policy.md
```

### Searching the Knowledge Base

```bash
# Search for documents about a topic
blot --openai-key=$(cat ./openai.key) \
  search "how is backups handled"

# Search and show the content of the documents
blot --openai-key=$(cat ./openai.key) \
  search --emit "what is our password policies"

# Limit the search results by label
blot --openai-key=$(cat ./openai.key) \
  search --limit=policies:3 --limit=procedures:2 "access control for consultants"
```

### Asking Questions

```bash
export BLOT_OPENAI_KEY=$(cat ./openai.key)
# Ask a question about the knowledge base
blot prompt what is our policy on remote work?

# Use a custom system prompt
blot prompt --system-prompt="You are a helpful assistant." what is our vacation policy?
```

### Filling CSV Data

```bash
export BLOT_OPENAI_KEY=$(cat ./openai.key)
# Fill a CSV file with data from the knowledge base
blot fill --in=input.csv --out=output.csv --delimiter=","
```

### Exploding a CSV into Individual Files

```bash
export BLOT_OPENAI_KEY=$(cat ./openai.key)
# Explode a CSV into individual files
blot explode --with-headers --delimiter="," data.csv

# Specify output directory
blot explode --out=./exploded_files --with-headers data.csv
```

## Development

Blot is built using Go and relies on several dependencies:
- [Bellman](https://github.com/modfin/bellman) for LLM/embedding model integration

