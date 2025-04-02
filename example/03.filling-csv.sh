#!/bin/bash


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
