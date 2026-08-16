package mcp

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// GitHubDockerTransport builds a stdio transport that launches the official
// GitHub MCP server (ghcr.io/github/github-mcp-server) as a short-lived
// Docker container per session. This mirrors the officially documented way
// to run the server locally:
//
//	docker run -i --rm -e GITHUB_PERSONAL_ACCESS_TOKEN ghcr.io/github/github-mcp-server
//
// The GitHub MCP server is stdio-only in its local/self-hosted form, so it
// is not a fit for a long-running docker-compose HTTP service; it is
// designed to be spawned by the MCP client itself. toolsets may be empty to
// use the server's default toolset ("context,repos,issues,pull_requests,users").
func GitHubDockerTransport(token string, toolsets []string) (gosdkmcp.Transport, error) {
	if token == "" {
		return nil, errors.New("mcp: github token is required")
	}

	args := []string{"run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN"}
	if len(toolsets) > 0 {
		args = append(args, "-e", "GITHUB_TOOLSETS")
	}
	args = append(args, "ghcr.io/github/github-mcp-server")

	cmd := exec.Command("docker", args...)
	cmd.Env = append(os.Environ(), "GITHUB_PERSONAL_ACCESS_TOKEN="+token)
	if len(toolsets) > 0 {
		joined := ""
		var joinedSb37 strings.Builder
		for i, t := range toolsets {
			if i > 0 {
				joinedSb37.WriteString(",")
			}
			joinedSb37.WriteString(t)
		}
		joined += joinedSb37.String()
		cmd.Env = append(cmd.Env, "GITHUB_TOOLSETS="+joined)
	}

	return &gosdkmcp.CommandTransport{Command: cmd}, nil
}

// QdrantHTTPTransport builds a streamable-HTTP transport pointing at a
// running qdrant-mcp-server instance (github.com/mhalder/qdrant-mcp-server),
// e.g. started in docker-compose with TRANSPORT_MODE=http. url should
// include the "/mcp" path, e.g. "http://qdrant-mcp:3000/mcp".
func QdrantHTTPTransport(url string) (gosdkmcp.Transport, error) {
	if url == "" {
		return nil, errors.New("mcp: qdrant mcp server url is required")
	}
	return &gosdkmcp.StreamableClientTransport{Endpoint: url}, nil
}
