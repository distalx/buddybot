# buddybot

A Slack bot for Codebuddies

We use a development Slack workspace to avoid noise in active Slack communities. If you'd like to contribute, create an issue providing a description of what you'd like to work on and we'll get you an invite.

Slack Workspace: [buddybotdev.slack.com](https://buddybotdev.slack.com/)

## Getting Started

Buddybot runs in a Docker container for ease of deployment. A makefile makes it easy to build (and run) the bot locally. This assumes you have Docker installed.

### Build

```plain
make build
```

### Run

You need to provide the Slack API token as the `BUDDYBOT_TOKEN` environment variable. By default we take your local environment variable and expose it inside the Docker container.

```plain
export BUDDYBOT_TOKEN="xoxb-xxxxx-xxxxx-xxxxx"
make run
```
