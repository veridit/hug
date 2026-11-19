---
id: p87ngv
title: "Inline marks on a task's text; resolved dialogue moves to a history file"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-19
updated: 2026-11-19
stage: "design"
design_since: "2026-11-19T09:56"
design_asked_by: "crab"
design_question: "Should questions and requests both become threads, so the ## Questions and ## Requests sections are replaced (old ones migrated), or do threads sit beside them?"
design_for: "jhf"
---

## Goal
The ticket stays the crisp working document. A person or agent marks a piece of a task's text with a comment, question or request that sits inline in the text, visible as text, with a recognizable sigil. The agent picks marks up, acts or answers, and resolves each by rewriting the ticket text to the clarified result and removing the mark. The dialogue that led there (the mark, replies, who and when) is appended to a history file attached to the task, never kept in the ticket.

## Purpose
Jørgen 2026-11-19 08:55Z: attach a request to a piece of the text in a mark mode, so the AI knows exactly what it is about; questions today look like acceptance criteria. 09:05Z (answer 1 on this task): threads inside the ticket would bloat it, and the conversation must not become the thing. The ticket must stay crisp: clarity comes from boiling text down to its essence. The messy part goes to a separate, suitably named history file in the task's files (provenance, chat history) that grows by appending.

## Principle
Ticket = working document, short and current. History = append-only provenance file, linked from the ticket, never needed to understand the ticket. Marks are plain text in the field they are about, so git, grep, any editor and any agent see them. Resolving a mark means changing the ticket, not adding to it. The agent instruction: lead with clarity in the ticket, answer and act on marks, post new questions as marks, move the dialogue to history. Reuse an existing markup convention for the sigil if one fits. Either side may resolve a mark (Jørgen 09:05Z): the person accepts, approves or asks for something else; the agent judges by the ticket whether the mark is answered. Both use judgment, and either can initiate a resolve for clarity.

## Relevant details
Supersedes the 08:56Z thread sketch (threads in ## Threads, rejected by Jørgen 09:05Z). Replaces ## Questions and ## Requests over time (q2adr8, hbgybn): open marks show in ASKED / requests as those do now; delivery as gmzd87. History file name, format and sigil are open design questions on this task. Candidate sigil: CriticMarkup ({==marked text==}{>>jhf: comment<<}), an existing plain-text review convention. TUI: a mark mode in the detail view selects a span; c comment, ? question, r request; open marks are highlighted calmly (qh9ep1 check 9) and a key jumps between them.

## Depends on

## Acceptance
- [ ] ~~Jørgen approves a mock (CLI transcript plus TUI frames at 100x30) of a thread on a quoted sentence, a reply and a resolve~~ (not relevant: threads in the ticket rejected by Jørgen 09:05Z; replaced by inline-mark checks, crab, 2026-11-19)
- [ ] ~~An outdated anchor (quoted text edited away) keeps its thread and shows it as outdated (test)~~ (not relevant: threads in the ticket rejected by Jørgen 09:05Z; replaced by inline-mark checks, crab, 2026-11-19)
- [ ] ~~Several questions in one call become separate threads, each answered on its own (test)~~ (not relevant: threads in the ticket rejected by Jørgen 09:05Z; replaced by inline-mark checks, crab, 2026-11-19)
- [ ] ~~Jørgen marks a sentence in hug tui and the assignee answers in that thread without a DM~~ (not relevant: threads in the ticket rejected by Jørgen 09:05Z; replaced by inline-mark checks, crab, 2026-11-19)
- [ ] Jørgen approves a mock (task file before and after, CLI transcript, TUI frames at 100x30): a sentence marked with a request, the agent's change, the mark gone from the ticket and the exchange in the history file
- [ ] Resolving a mark rewrites or keeps the marked text, removes the mark, and appends mark, replies, who and when to the history file; the ticket holds no dialogue afterwards (test)
- [ ] Several questions in one call become separate inline marks, each answered on its own (test)
- [ ] An open mark shows in the person's ASKED or the assignee's requests and is delivered as today (gmzd87) (test)
- [ ] The agent instructions say: lead with clarity in the ticket, act on or answer marks, post questions as marks, move the dialogue to the history file (fd0j4p wording)
- [ ] Jørgen marks a sentence in hug tui, the assignee resolves it, and the ticket reads clearer without a DM

## Questions
- 2026-11-19T09:56 question 1 by crab to jhf:
  Should questions and requests both become threads, so the ## Questions and ## Requests sections are replaced (old ones migrated), or do threads sit beside them?
- 2026-11-19T09:56 question 2 by crab to jhf:
  Who may resolve a thread: only the person who opened it, or also the agent once it has acted on it?
  answered 2026-11-19T10:06 by jhf:
    (from jhf's request 1, 09:05Z, written because he could not reach question 2 in hug tui) Both may resolve. It depends on what the request and the answer say: the person may accept or approve, or say they need something else; the agent decides by the ticket too. Use judgment, and either can initiate a resolve for clarity.
- 2026-11-19T09:56 question 3 by crab to jhf:
  Besides task text, should threads anchor to an attached file such as a mock sheet (a region or a line in it)?
- 2026-11-19T10:05 question 4 by crab to jhf:
  Sigil: should marks use CriticMarkup, an existing plain-text review convention: {==marked text==}{>>jhf: why this?<<} for a comment or question, with a request tagged {>>jhf request: shorten<<}? Or do you want a hug-specific sigil?
- 2026-11-19T10:05 question 5 by crab to jhf:
  History file: one append-only history.md per task in its files (.hug/tasks/p/p87ngv/history.md), linked from the ticket, with each resolved mark as a dated entry quoting the original text, the exchange and the change made. Is that the right name and shape?

## Requests
- 2026-11-19T10:05 request 1 by jhf:
  And now I can't get the second question here. I think your question number two both may resolve a thread. It depends on what the request says and the answer says. So the person, human who's doing this, will maybe say, ah, that's good. I accept or I approve. And it will say, no, I need something else. And you have to decide by the ticket. And I think the same by the agent. So we have to use judgment here. And both can initiate for clarity.

## Notes


crab 09:06Z: reshaped after Jørgen's answer 1 (09:05Z). Questions 1 and 3 were framed around threads; 1 is answered (no threads in the ticket, dialogue to a history file). Question 3 (anchor to an attached file) still stands in the new model. New questions 4 (sigil) and 5 (history file).
## History
- 2026-11-19T09:56 todo -> todo/design (ask)
- 2026-11-19T09:56 asked question 2 of jhf, 2 open (todo/design -> todo/design)
- 2026-11-19T09:56 asked question 3 of jhf, 3 open (todo/design -> todo/design)
- 2026-11-19T10:05 answered design, by jhf, for crab (todo/design -> todo/ready):
  > Should questions and requests both become threads, so the ## Questions and ## Requests sections are replaced (old ones migrated), or do threads sit beside them?
  I think it's a real risk that the conversation becomes the thing and I think it should be moved out to maybe a separate document that will link in the files section that is historic, almost like provenance or just chat history. Otherwise, the ticket will become bloated. The ticket has to stay crisp and clear. So I think the questions and requests might need replacement, but they might appear in the text so you can actually see them, just like text. We have to choose the sigils or how to recognize them and act on them. And then, as they are resolved, the ticket is changed, and any history or chat history is put into another document outside of the ticket. If we try to scram everything into the ticket, I think it will be like if you make threads inside the ticket, it will become bloated. And especially when things are complex, you will have longer and more intense chats, which will bloat them even more. When what you need is clarity. And clarity is often to find the succinctness in things and boil it down to its essence, not adding more text. So the text is the tool to which we reach short clarity very often. That's why the messy part has to go out into a different file. But I think the working document is the ticket itself. That's where the replies come, it's picked up, and then the AI gets an instruction to lead with clarity in the ticket, answer requests, do them, post any questions, and then put all of the prehistory provenance into a suitably named asset that can grow with a penned
- 2026-11-19T10:05 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-19T10:05 answer 1 acknowledged by crab
- 2026-11-19T10:05 crab read the answer: hug task set p87ngv
- 2026-11-19T10:05 asked question 4 of jhf, 4 open (todo/ready -> todo/design)
- 2026-11-19T10:05 asked question 5 of jhf, 5 open (todo/design -> todo/design)
- 2026-11-19T10:05 request 1 by jhf
- 2026-11-19T10:05 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-19T10:06 answered design (question 2), by jhf, for crab (todo/design -> todo/design):
  > Who may resolve a thread: only the person who opened it, or also the agent once it has acted on it?
  (from jhf's request 1, 09:05Z, written because he could not reach question 2 in hug tui) Both may resolve. It depends on what the request and the answer say: the person may accept or approve, or say they need something else; the agent decides by the ticket too. Use judgment, and either can initiate a resolve for clarity.
- 2026-11-19T10:06 answer 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-19T10:06 crab read the answer: hug task set p87ngv
- 2026-11-19T10:06 request 1 acknowledged by crab
