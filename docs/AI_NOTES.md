# How I used AI

AI read the source pack, drafted the documents, wrote the Go POC and tests, and ran local checks. I set the scope and quality bar, asked for mathematical support, challenged assumptions, and directed revisions. The [command center](COMMAND_CENTER.md) was our first working artifact: it put the problem, decisions, and evidence in one place before we wrote the review, design, and code.

The work then moved in order through the [ranked review](REVIEW.md), [production design](DESIGN.md), tested allocator, mock partners, and [measured scenarios](../poc/README.md). I asked for a small, changeable POC rather than a new platform. Later I pushed for clearer code and prose across the submission and for a budget sweep that shows the latency and value tradeoff. The command center has the full task log.

## Where AI was wrong

The assistant first said an unviewed returned card could never be a negative click example. That was too broad. For **click per returned card at a position**, an unclicked return is a valid zero. For **click after actual display**, it is not. Writing the expected-value equation made the missing prediction target clear; we corrected the command center, review, and design.

An independent AI review also found that a reply arriving just after the deadline could enter selection, and that the first example did not show a lower-CPC winner. We added the deadline check, corrected the example, and reran tests and measurements. A later CLI test caught another issue: explicitly passing `-n=-1` silently used the config default. That now returns an error.

## Time spent

For the work visible in this chat, my time was roughly 70% directing, 30% reviewing and challenging, and 0% hand-writing prose or code. AI produced the drafts and ran the commands. This estimate excludes time outside the chat.
