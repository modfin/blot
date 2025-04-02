

# Example

This example is based on files in this directory and contains a fictive ISMS along with a few QA
associated with supplier assessments

## Install 

```bash 
go install github.com/modfin/blot@latest
```

## Get example

```bash
git clone https://github.com/modfin/blot.git
cd blot/example
```

## Embedding

Start of by picking one Embedding model you want to use.
- Voyage AI, https://docs.voyageai.com/docs/embeddings
- Open AI, https://platform.openai.com/docs/models#embeddings
- Vertex AI, https://cloud.google.com/vertex-ai/generative-ai/docs/learn/models#models


If you decide to use another embedding model later, you have to reprocess it, ie delete the database

Get the associated api key and model to your configuration

Default `--embed-model=OpenAI/text-embedding-3-small`


## API Keys

Once you select your models, retrieve the api keys for them to be used with blot.

eg `echo sk-proj... > ./openai.key`

> This varies by provider. Eg for google vertexai you need to create a service account and download the json file. see `blot --help` for providers

## Create the knowledge base

This is easily done by iterating through the documents you want to add

It can be ran iteratively,
and will only update embeddings for new documents or if content has changed.
Ie, run the script below once changes are made.

```bash
## Add all of our ISO 27001 policies
# under the label "policies"
blot --verbose \
   --openai-key=$(cat ./openai.key) \
   --embed-model="OpenAI/text-embedding-3-small" \
     add --label=policies \
     ./knowledge/policies/*.md

## Pre-processing csv file by exploding it into one file per row
blot --verbose \
   explode \
     --with-headers \
     --delimiter="," \
     --out=./knowledge/question-n-answers \
       ./knowledge/questions-and-answers.csv

## Adding all fragments/rows from the csv file into the knowledge base
# under the label "QA"
blot --verbose \
   --openai-key=$(cat ./openai.key) \
   --embed-model="OpenAI/text-embedding-3-small" \
     add --label=QA \
     ./knowledge/question-n-answers/*_questions-and-answers.csv


```


## Searching the knowledge base after relevant documents

```bash
# Search for documents about a topic
blot --verbose \
   --openai-key=$(cat ./openai.key) \
   --embed-model="OpenAI/text-embedding-3-small" \
     search --limit 1 --emit \
     "how is backups handled"
     
# knowledge/question-n-answers/0030_questions-and-answers.csv
# Question:       How do you ensure the integrity and confidentiality of backups?
# Answer: Backups are encrypted both in transit and at rest. 
# Access to backups is restricted to authorized personnel, and backup integrity is 
# regularly tested through restoration exercises.
```

## LLM

Once we have knowledge base, we can use it to answer questions which means we need an LLM.

Default `--llm-model=OpenAI/gpt-4o-mini`

Picking the LLM model you want to use and an associated api key, just as for embeddings.

- Anthropic https://docs.anthropic.com/en/docs/about-claude/models/all-models
- OpenAI https://platform.openai.com/docs/models
- VertexAI https://cloud.google.com/vertex-ai/generative-ai/docs/learn/models


## Prompting the knowledge base after relevant documents
```bash 
# Prompting the knowledge base is essentially using RAG and picking documents from the
# knowledge base and passing them to an LLM to answer your question
blot --verbose \
   --openai-key=$(cat ./openai.key) \
   --embed-model="OpenAI/text-embedding-3-small" \
   --llm-model="OpenAI/gpt-4o-mini" \
     prompt \
      --limit=QA:5 --limit=policies:2 \
      --system-prompt="You are a CISO who answers ISO 27001 supplier assessment question" \
     "Do you have a formal business continuity plan and disaster recovery procedures? How often are they tested?"
     
# Yes, we have a documented Business Continuity Plan (BCP) and Disaster Recovery Plan (DRP).
# These plans are tested annually to ensure their effectiveness and to keep our team familiar 
# with the procedures.

```

## Filling a CSV with answers from the knowledge base

Many supplier assessments and questioners are in a Row based such as Excel or CSV.
The fill tool adds to columns to the end of every row,
a answer from the knowledge base and a confidence score

```bash 
blot --verbose \
   --openai-key=$(cat ./openai.key) \
   --embed-model="OpenAI/text-embedding-3-small" \
   --llm-model="OpenAI/gpt-4o-mini" \
   fill \
     --limit=policies:2 --limit=QA:3 \
     --delimiter="\\t" \
     --system-prompt="You are a CISO who answers ISO 27001 supplier assessment question. Answer short, a sentence or two" \
     --in=./supplier-assesment.tsv \
     --out=./supplier-assesment-answers.tsv
     
# Q5	How do you protect data at rest, in transit, and in use, particularly when handling our organization's data?
# A: We protect data at rest using AES-256 encryption, in transit with TLS 1.3 encryption, and in use through strict access controls and monitoring mechanisms to ensure only authorized personnel can handle the data.	0.900
#
# Q6	What is your data retention and disposal policy, and how do you ensure secure deletion of our data upon contract termination?
# A: We have a Data Retention Policy that specifies retention periods based on legal and business needs, and upon contract termination, we securely delete your data using methods such as cryptographic erasure or physical destruction, while maintaining records of the disposal process.	0.950
#
# Q7	Do you have a formal business continuity plan and disaster recovery procedures? How often are they tested?
# A: Yes, we have a formal Business Continuity Plan (BCP) and Disaster Recovery Plan (DRP) that are tested annually.	0.950

```


