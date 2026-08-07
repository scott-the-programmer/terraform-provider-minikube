# Container runtime for the dependabot-triage skill.
#
# Ships with: claude-code CLI, gh, git, make, jq, docker CLI (uses host socket).
# Creds (gh token, claude session, gitconfig) are mounted at run time — never baked in.

FROM node:22-bookworm-slim

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates curl gnupg git make jq openssh-client \
 && install -m 0755 -d /etc/apt/keyrings \
 # gh CLI repo
 && curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
      | gpg --dearmor -o /etc/apt/keyrings/githubcli.gpg \
 && chmod go+r /etc/apt/keyrings/githubcli.gpg \
 && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli.gpg] https://cli.github.com/packages stable main" \
      > /etc/apt/sources.list.d/github-cli.list \
 # docker CLI repo (client only — uses host daemon via mounted socket)
 && curl -fsSL https://download.docker.com/linux/debian/gpg \
      | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
 && chmod go+r /etc/apt/keyrings/docker.gpg \
 && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian bookworm stable" \
      > /etc/apt/sources.list.d/docker.list \
 && apt-get update \
 && apt-get install -y --no-install-recommends gh docker-ce-cli \
 && rm -rf /var/lib/apt/lists/*

# Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code

# Non-interactive git
ENV GIT_TERMINAL_PROMPT=0

ENTRYPOINT ["claude"]
