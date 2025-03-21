# gtree

[![Website](https://img.shields.io/badge/Website-gtree.henil.dev-brightgreen)](https://gtree.henil.dev) [![Demo](https://img.shields.io/badge/Demo-gtree.henil.dev/henilmalaviya/gtree-blue)](https://gtree.henil.dev/henilmalaviya/gtree) [![GitHub](https://img.shields.io/github/stars/henilmalaviya/gtree?style=social)](https://github.com/henilmalaviya/gtree)

## Overview

**gtree** is a service that generates a text-based tree representation of GitHub repositories. It’s designed to help developers and AI models like LLMs quickly understand a project’s structure by providing a concise, organized view of the repo contents.

Live Demo: [gtree.henil.dev](https://gtree.henil.dev)

### Features

- Text-based tree representation of repositories
- Shareable URL
- Mimics the Linux `tree` command
- Great for feeding into LLMs to help understand project structure

## Self-Hosting

If you want to run gtree locally:

```bash
# Clone the repo
git clone https://github.com/henilmalaviya/gtree.git

# Install dependencies
bun install

# Run locally
bun run start
```

### Environment Variables

To avoid hitting GitHub’s rate limits, you can provide a GitHub token:

```
GITHUB_TOKEN=your_personal_token_here
```

This allows gtree to make authenticated requests to the GitHub API, increasing the rate limit.

## Contributing

Got ideas for improvements? Feel free to open an issue or a pull request!
