#!/bin/bash

## This file can be ran multiple times to create the knowledge base.
# It will only replaced things that has been changed.

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
