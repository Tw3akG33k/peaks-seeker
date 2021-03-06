package provider

import (
	"net/url"

	"github.com/google/go-github/v29/github"
)

func makeClient(baseURL string) *github.Client {
	cli := github.NewClient(nil)
	cli.BaseURL, _ = url.Parse(baseURL)
	return cli
}
