# Contributing to gvcode

We appreciate your interest in contributing to gvcode. Before you start, please read the following guidelines to ensure a smooth collaboration.

## How to Contribute

### Submitting Issues
When opening an issue, please follow these guidelines:

- Check if it’s already reported. 
- Include clear steps to reproduce the problem. 
- Feature requests are welcome but should be well-defined.

### Pull Requests

Before submitting a PR, please follow these guidelines:

- Check for open PRs to avoid duplicate work.
- Follow the project's coding style (see below).
- Keep PRs focused – small, well-scoped changes are easier to review.
- Write clear commit messages describing what the PR does.
- If your PR fixes an issue, reference it in the description (e.g., Fixes #42).

#### What PRs Will Be Rejected?

We appreciate all contributions, but some PRs may be closed if they:

- Make unnecessary stylistic changes (e.g., reordering imports, renaming variables without strong reason).
-  Reformat code without a functional improvement.
-  Modify large parts of the codebase without prior discussion.
-  Introduce performance regressions or unnecessary complexity.

## Coding Standards

Code formatting: Follow gofmt for Go code.
Linting: Run golangci-lint run before submitting a PR.
Performance considerations: Ensure changes do not introduce unnecessary allocations or degrade performance.
Error handling: Always check and return errors properly.

## Meaningful Ways to Contribute

Here are some good ways to help:

- Fix bugs (check the "good first issue" tag).
- Optimize performance (profiling is encouraged!).
- Improve documentation (clarify usage, add examples).
- Add test cases for untested functionality.
- Suggest new features with proper justification.



Thanks for contributing! 