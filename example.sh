#!/bin/bash


## Start of by picking one Embedding model you want to use.
#  - Voyage AI, https://docs.voyageai.com/docs/embeddings
#  - Open AI, https://platform.openai.com/docs/models#embeddings
#  - Vertex AI, https://cloud.google.com/vertex-ai/generative-ai/docs/learn/models#models
#
# If you decide to use other embedding model later, you have to reprocess it.
## Add the associated api key and model to your configuration
#   eg. `blot --openai-key=$(cat ./openai.key) --embed-model=OpenAI/text-embedding-3-small ...`


## Default
### --embed-model=OpenAI/text-embedding-3-small
### --llm-model=OpenAI/gpt-4o-mini



## Creating the knowledge base

### Start off by prepping the knowledge base, simply put all files into a folder structure that works.

## Create a script that creates the knowledge base
### This can later be used to update the knowledge base when information is added

## Adding files to the knowledge base with label policies
go run main.go --verbose --openai-key=$(cat ./files/openai.key) \
   add --label=policies \
    ./files/policies/*.md

## Adding files to the knowledge base with label procedures
go run main.go --verbose --openai-key=$(cat ./files/openai.key) \
   add --label=procedures \
    ./files/procedures/*.md


## Exploding a tsv file into separate files for each line
### blot really only handles singe files, so the eaiest way to handle csv, excel and such
### is to explode it into separate files
go run main.go --verbose \
   explode --with-headers --delimiter="\\t" \
    ./files/question-n-answers.tsv

## Adding to the knowledge base files with label QA
go run main.go  --verbose --openai-key=$(cat ./files/openai.key) \
   add --label=QA \
    ./files/question-n-answers.tsv.exploded/*_question-n-answers.tsv


## Searching for relevant documents
## Returns references 2 policies and 3 QA
go run main.go --verbose --openai-key=$(cat ./files/openai.key) \
   search
     --limit=policies:2 --limit=QA:3 \
      "How is backups handled and what RPO/RTO objectives"

## Searching for relevant documents and open it in pager
## Returns the document of 2 policies and 3 QA
go run main.go --verbose --openai-key=$(cat ./files/openai.key) \
   search
     --limit=policies:2 --limit=QA:3 --emit \
      "How is backups handled and what RPO/RTO objectives" \
        | less



## Searching for relevant documents
## Returns references 2 policies and 3 QA
go run main.go --verbose --openai-key=$(cat ./files/openai.key) \
   prompt \
     --limit=policies:2 --limit=QA:3 \
     --system-prompt="You are CISO and are answering a supplier assessment. Only answer with at most 3 sentences" \
      "How is backups handled and what are the RPO/RTO objectives"