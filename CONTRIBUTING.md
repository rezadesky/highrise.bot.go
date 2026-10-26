# Contributing to highrisebot

Thank you for your interest in contributing to **highrisebot**!

## Disclaimer
> [!IMPORTANT]
> `highrisebot` is an **unofficial**, community-driven SDK. It is not affiliated with, endorsed, or sponsored by Pocket Worlds or Highrise Virtual Reality.

## How to Contribute
1. **Fork** the repository on GitHub.
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/highrisebot.git
   cd highrisebot
   ```
3. Create a feature branch:
   ```bash
   git checkout -b feat/my-new-feature
   ```
4. Run tests and verify formatting:
   ```bash
   go test -v -race ./...
   ```
5. Commit your changes following [Conventional Commits](https://www.conventionalcommits.org/):
   ```bash
   git commit -m "feat: add support for new room event"
   ```
6. Push to your branch and submit a **Pull Request**.

## Reporting Bugs
- Please check existing issues before submitting a new one.
- Provide a minimal reproducible example, Go version, and room error payload if applicable.
- **Never post your bot token or private room credentials in public issues.**
