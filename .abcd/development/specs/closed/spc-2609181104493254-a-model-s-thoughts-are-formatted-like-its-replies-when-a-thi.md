---
id: spc-2609181104493254
slug: a-model-s-thoughts-are-formatted-like-its-replies-when-a-thi
intent: itd-2609181104497297
origin: researcher-authored
production_mode: hand-written
---
# A model's thoughts are formatted like its replies

## Summary

The Thoughts row draws its text through the same MarkdownBlocks the reply
uses, kept in the row's state and re-parsed on change with the reply's
throttle; the blocks are drawn by the same block view, in the callout
italic secondary style the row has today.

## Scope

In scope: MessageRow in client/GropiusChat/GropiusChat.swift — a shared
block view used by the reply and the thoughts, a reasoningBlocks state with
the same scheduling as blocks; an architecture test. Out of scope: effects
on thoughts; the Copy item (unchanged, copies the reply's text).

## Approach

A blockViews(_:animate:) builder replaces the inline switch in reply; the
Thoughts row calls it with animate false over reasoningBlocks, parsed in
onAppear and on change of displayReasoning through scheduleParse's twin.

## How the acceptance criteria are met

1. Italic and bold in thoughts — the parser and Text's inline intents.
2. The test reads reasoningBlocks and the parse call in reasoningDisclosure.

## Verification

Suite, build, docs lint; checked by hand with a thinking model.
