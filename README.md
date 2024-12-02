# Gitea Repo Syncer

Gitea Repo Syncer is a tool that mirror your all github starred repositories into a private [gitea](https://gitea.io) server. It also can sync repository owners by auto creating users and organizations.

## Features

- 🔄 Automatically sync starred GitHub repositories to Gitea
- 👥 Auto-create users and organizations in Gitea
- 🔁 Keep repositories synchronized with original GitHub sources
- 🔐 Support for private Gitea instances
- ⚡ Efficient mirroring process

## Prerequisites

- A running Gitea instance
- GitHub account with starred repositories

## Usage

The simplest way to run the syncer is using command-line flags:

```bash
./gitea-repo-syncer sync-github-starred-repos \
--gitea-server-url="https://your-gitea.com" \
--gitea-user="your-gitea-username" \
--gitea-auth-token="your-gitea-token" \
--github-user="your-github-username" \
--github-auth-token="your-github-token"
```

You can also set environment variables instead of using command-line flags:

```bash
export GITEA_SERVER_URL="https://your-gitea.com"
export GITEA_USER="your-gitea-username"
export GITEA_AUTH_TOKEN="your-gitea-token"
export GITHUB_USER="your-github-username"
export GITHUB_AUTH_TOKEN="your-github-token"
./gitea-repo-syncer sync-github-starred-repos
```

## Build

1. Clone this repository:

    ```bash
    git clone https://github.com/yourusername/gitea-repo-syncer.git
    cd gitea-repo-syncer
    ```

2. Build the project:

    ```bash
    make
    ```

## TODO

### Core Features
- [ ] GitHub
  - [x] Github starred repositories sync
  - [ ] Github forked repositories sync
  - [ ] Github source repositories sync
  - [ ] Github starred gist sync
  - [ ] Github source gist sync
- [ ] GitLab
- [ ] Gitea

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Gitea](https://gitea.io) for providing an excellent self-hosted Git service
- [github-gitea-mirror](https://github.com/varunsridharan/github-gitea-mirror): Simple Python Script To Mirror Repository / Gist From Github To Gitea
- All contributors who help improve this project
