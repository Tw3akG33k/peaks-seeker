package provider

import (
	"sync"
	"context"
	"time"
	"math/rand"

	log	"github.com/sirupsen/logrus"

	"github.com/lucmichalski/peaks-seeker/pkg/client"
)

func ptrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrBool(b *bool) bool {
	return b != nil && *b
}

type IntRange struct {
	min, max int
}

// get next random value within the interval including min and max
func (ir *IntRange) NextRandom(r *rand.Rand) int {
	return r.Intn(ir.max-ir.min+1) + ir.min
}

func checkForRemainingLimit(ghc *client.GHClient, ghcm *client.ClientManager, isCore bool, minLimit int, debug bool) {

	var (
		wg          sync.WaitGroup
		limit, rate int
	)
	if debug {
		log.Printf("checkForRemainingLimit, isCore=%t, minLimit=%d ", isCore, minLimit)
	}

getRate:
	if debug {
		log.Println("checkForRemainingLimit.rateLimits")
	}
	rateLimits, resp, err := ghc.Client.RateLimits(context.Background())
	if err != nil {
		if debug {
			log.Warnf("could not access rate limit information: %s\n", err)
		}
		goto changeClient
	}

	if isCore {
		rate = rateLimits.GetCore().Remaining
		limit = rateLimits.GetCore().Limit
	} else {
		rate = rateLimits.GetSearch().Remaining
		limit = rateLimits.GetSearch().Limit
	}

	if debug {
		log.Printf("checkForRemainingLimit, rate=%d, limit=%d, minLimit=%d ", rate, limit, minLimit)
	}

	if rate < minLimit {
		if debug {
			log.Printf("Not enough rate limit: %d/%d/%d\n", rate, minLimit, limit)
		}
		r := rand.New(rand.NewSource(55))
		ir := IntRange{10, 60}
		<-time.After(time.Second * time.Duration(ir.NextRandom(r)))
		goto changeClient
	}

	if debug {
		log.Printf("Rate limit: %d/%d\n", rate, limit)
	}
	return

changeClient:
	{
		if debug {
			log.Warnln("checkForRemainingLimit.changeClient...")
		}
		go func() {
			wg.Add(1)
			defer wg.Done()
			if debug {
				log.Warnln("checkForRemainingLimit.ghc.Reclaim...")
			}
			client.Reclaim(ghc, resp)
		}()

		if debug {
			log.Warnln("checkForRemainingLimit.ghcm.Fetch...")
		}
		ghc = ghcm.Fetch()
		goto getRate
	}

}
