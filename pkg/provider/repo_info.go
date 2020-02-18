package provider

import (
	"fmt"
	"context"
	"time"
	"sync"
	"strings"

	"go.uber.org/zap"
	log	"github.com/sirupsen/logrus"
	"github.com/google/go-github/v29/github"
	"github.com/lucmichalski/peaks-seeker/pkg/data"
	"github.com/lucmichalski/peaks-seeker/pkg/client"
)

const (
	itemsPerPage   = 100
	itemsPerSearch = 1000
)

// RepoInfos returns (almost) all repositories given the selected criteria via a channel
func RepoInfos(
	// cli *github.Client,
	ghc *client.GHClient, 
	ghcm *client.ClientManager,
	language string,
	starMin int,
	starMax int,
	repoInfoCh chan *data.RepoInfo,
	doneCh chan struct{},
) error {

	// ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	// defer cancel()
	var (
		// client *ghclient.GHClient
		ok      bool
		wg      sync.WaitGroup
		res 	*github.RepositoriesSearchResult
		resp    *github.Response
		e       *github.AbuseRateLimitError
		err  	error
	)

getSearch:	
	for cursor := starMin; cursor < starMax; {
		start := cursor
		end := start + (start / 10)
		cursor = end

		query := fmt.Sprintf("language:%s stars:%d..%d", language, start, end)

		opts := &github.SearchOptions{
			Sort:        "updated",
			ListOptions: github.ListOptions{
				PerPage: itemsPerPage,
			},
		}

		for page := 1; page <= itemsPerSearch/itemsPerPage; page++ {
			opts.ListOptions.Page = page
			res, resp, err = ghc.Client.Search.Repositories(context.Background(), query, opts)
			checkForRemainingLimit(ghc, ghcm, false, 5, true)
			log.Infoln(resp.Request.URL.String())
			if err != nil {
				if _, ok = err.(*github.RateLimitError); ok {
					log.Error("RepoInfos hit limit error, it's time to change client.", zap.Error(err))

					goto changeClient
				} else if e, ok = err.(*github.AbuseRateLimitError); ok {
					log.Error("RepoInfos have triggered an abuse detection mechanism.", zap.Error(err))
					time.Sleep(*e.RetryAfter)
					goto getSearch

				} else if strings.Contains(err.Error(), "timeout") {
					log.Info("RepoInfos has encountered a timeout error. Sleep for five minutes.")
					time.Sleep(5 * time.Minute)
					goto getSearch

				} else {
					log.Error("RepoInfos terminated because of this error.", zap.Error(err))
					return err
				}
			}

			for _, repo := range res.Repositories {
				repoInfo := makeRepoInfo(ghc, ghcm, &repo)
				if repoInfo != nil {
					repoInfoCh <- repoInfo
				}
			}

			if len(res.Repositories) < itemsPerPage {
				break
			}
		}
	}
	doneCh <- struct{}{}

changeClient:
	{
		log.Warnln("RepoInfos.changeClient...")
		go func() {
			wg.Add(1)
			defer wg.Done()
			client.Reclaim(ghc, resp)
		}()
		ghc = ghcm.Fetch()
		goto getSearch
	}

	return nil


}

func makeRepoInfo(ghc *client.GHClient, ghcm *client.ClientManager, repo *github.Repository) *data.RepoInfo {

	// ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	// defer cancel()
	var (
		// client *ghclient.GHClient
		ok      bool
		wg      sync.WaitGroup
		resp    *github.Response
		langs   map[string]int
		e       *github.AbuseRateLimitError
		err  	error
	)

getRepo:
	{
		info := &data.RepoInfo{
			Owner:         *repo.Owner.Login,
			Name:          *repo.Name,
			Stars:         *repo.StargazersCount,
			Contributions: make(map[string]int),
		}
		// pp.Println(info)

		langs, resp, err = ghc.Client.Repositories.ListLanguages(context.Background(), info.Owner, info.Name)
		checkForRemainingLimit(ghc, ghcm, true, 5, true)
		log.Infoln(resp.Request.URL.String())
		if err != nil {
			if _, ok = err.(*github.RateLimitError); ok {
				log.Error("makeRepoInfo hit limit error, it's time to change client.", zap.Error(err))

				goto changeClient
			} else if e, ok = err.(*github.AbuseRateLimitError); ok {
				log.Error("makeRepoInfo have triggered an abuse detection mechanism.", zap.Error(err))
				time.Sleep(*e.RetryAfter)
				goto getRepo

			} else if strings.Contains(err.Error(), "timeout") {
				log.Info("makeRepoInfo has encountered a timeout error. Sleep for five minutes.")
				time.Sleep(5 * time.Minute)
				goto getRepo

			} else {
				log.Error("makeRepoInfo terminated because of this error.", zap.Error(err))
				return nil
			}
		}

		info.LangPercentages = makeLangPercentages(langs)
		if info.LangPercentages == nil {
			// GitHub cannot detect languages -> ignore this repo
			return nil
		}

		contributors, resp, err := ghc.Client.Repositories.ListContributors(context.Background(), info.Owner, info.Name, &github.ListContributorsOptions{},)
		checkForRemainingLimit(ghc, ghcm, true, 5, true)
		log.Infoln(resp.Request.URL.String())
		if err != nil {
			if _, ok = err.(*github.RateLimitError); ok {
				log.Error("fetchGlobalTopics hit limit error, it's time to change client.", zap.Error(err))

				goto changeClient
			} else if e, ok = err.(*github.AbuseRateLimitError); ok {
				log.Error("fetchGlobalTopics have triggered an abuse detection mechanism.", zap.Error(err))
				time.Sleep(*e.RetryAfter)
				goto getRepo

			} else if strings.Contains(err.Error(), "timeout") {
				log.Info("fetchGlobalTopics has encountered a timeout error. Sleep for five minutes.")
				time.Sleep(5 * time.Minute)
				goto getRepo

			} else {
				log.Error("fetchGlobalTopics terminated because of this error.", zap.Error(err))
				return nil
			}
		}

		for _, contrib := range contributors {
			info.Contributions[*contrib.Login] = *contrib.Contributions
		}
		return info
	}

changeClient:
	{
		log.Warnln("getTopics2.changeClient...")
		go func() {
			wg.Add(1)
			defer wg.Done()
			client.Reclaim(ghc, resp)
		}()
		ghc = ghcm.Fetch()
		goto getRepo
	}

}

func makeLangPercentages(langBytes map[string]int) map[string]float64 {
	percentages := make(map[string]float64)

	totalBytes := 0
	for _, bytes := range langBytes {
		totalBytes += bytes
	}
	if totalBytes == 0 {
		return nil
	}

	for lang, bytes := range langBytes {
		percentages[lang] = float64(bytes) / float64(totalBytes)
	}

	return percentages
}
