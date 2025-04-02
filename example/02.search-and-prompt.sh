#!/bin/bash


## Searching
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


## Prompting
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