package main

import (
	"fmt"
	"strconv"
	"strings"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/lucmichalski/peaks-seeker/pkg/data"
	"github.com/lucmichalski/peaks-seeker/pkg/client"
	"github.com/lucmichalski/peaks-seeker/pkg/provider"
	"github.com/lucmichalski/peaks-seeker/pkg/score"
)

const (
	repoMaxStars = 150000
	cacheDir     = "cache"
)

func main() {
	// init env values from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// github client init with multiple tokens
	tokens := strings.Split(os.Getenv("PEAKS_GITHUB_TOKENS"),",")
	clientManager := client.NewManager(cacheDir, tokens)
	defer clientManager.Shutdown()
	clientGH := clientManager.Fetch()

	// init score state
	scores := score.NewState()

	// get the number of channels
	parallelJobs, err := strconv.Atoi(os.Getenv("PEAKS_PARALLEL_JOBS"))
	if err != nil {
		log.Fatal(err)
	}
	repoInfoCh := make(chan *data.RepoInfo, parallelJobs)
	doneCh := make(chan struct{})

	// locations
	data.SetLocations(os.Getenv("PEAKS_LOCATIONS"))

	// languages
	languages := strings.Split(os.Getenv("PEAKS_LANGUAGES"), "|")
	for _, lang := range languages {
		go provider.RepoInfos(clientGH, clientManager, lang, repoMinStars(), repoMaxStars, repoInfoCh, doneCh)

	fetch:
		for {
			select {
			case <-doneCh:
				break fetch
			case repoInfo := <-repoInfoCh:
				scores.AddRepoInfo(repoInfo)
			}
		}
	}

	users := scores.GetUsersRanked()

	for _, user := range users {
		if !user.ScoreOK() {
			continue
		}
		provider.GetUserInfo(clientGH, clientManager, user)
		if !user.DataOK() {
			continue
		}
		fmt.Printf("%s\t%s\t%d\t%s\t%s\t%s\t%s\t%t\n",
			user.Username, user.Name, int(user.Score), user.LanguageInfo,
			user.Location, user.Company, user.Email, user.Hireable,
		)
	}
}

func repoMinStars() (num int) {
	num, _ = strconv.Atoi(os.Getenv("PEAKS_REPO_MIN_STARS"))
	if num < 1 {
		panic("PEAKS_REPO_MIN_STARS too small!")
	} else if num >= repoMaxStars {
		panic("PEAKS_REPO_MIN_STARS too large!")
	}
	return
}
