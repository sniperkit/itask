package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cnf/structhash"
	"github.com/golangplus/errors"

	sizedwaitgroup "github.com/sniperkit/xutil/plugin/concurrency/sync/sized"
	cmap "github.com/sniperkit/xutil/plugin/map/multi"

	// cmap "github.com/fanliao/go-concurrentMap"
	//sync "github.com/sniperkit/xutil/plugin/concurrency/sync/debug"

	"github.com/sniperkit/xvcs/plugin/provider/github/go-github/pkg"
	"github.com/tiaotiao/mapstruct"

	// "github.com/google/go-github/github"
	// "github.com/abourget/llerrgroup"

	// requests
	// "github.com/franela/goreq"

	// "github.com/k0kubun/pp"
	// "github.com/anacrolix/sync"
	// "github.com/viant/toolbox"
	// "github.com/thoas/go-funk"
	// "github.com/tuvistavie/structomap"
	// "github.com/src-d/enry/data"

	"github.com/sniperkit/xtask/util/runtime"
	"github.com/sniperkit/xutil/plugin/debug/pp"
	"github.com/sniperkit/xutil/plugin/format/convert/mxj/pkg"
	"github.com/sniperkit/xutil/plugin/struct"
)

// Analyzing trends on Github using topic models and machine learning.
// var wg sync.WaitGroup

func (g *Github) counterTrack(name string, incr int) {
	go func() {
		g.counters.Increment(name, incr)
	}()
}

func (g *Github) GetFunc(entity string, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	if g.Client == nil {
		return nil, nil, errInvalidClient
	}

	/*
		if ExceededRateLimit(g.client) {
			log.Debugln("new client required as exceeded rate limit detected for the current token, token.old=", g.ctoken, "debug", runtime.WhereAmI())
			g = g.Manager.Fetch()
		}
	*/

	switch entity {
	case "getTrends":
		return getTrends(g, opts)

	case "getReleaseTags":
		return getReleaseTags(g, opts)

	case "getCode":
		return getCode(g, opts)

	case "getStars":
		return getStars(g, opts)

	case "getRepoList":
		return getRepoList(g, opts)

	case "getUser":
		return getUser(g, opts)

	case "getUserOrgs":
		return getUserOrgs(g, opts)

	case "getUserNode":
		return getUserNode(g, opts)

	case "getFollowers":
		return getFollowers(g, opts)

	case "getFollowing":
		return getFollowing(g, opts)

	case "getRepo":
		return getRepo(g, opts)

	case "getReposByOrg":
		return getReposByOrg(g, opts)

	case "getReadme":
		return getReadme(g, opts)

	case "getTree":
		return getTree(g, opts)

	case "getLanguages":
		return getLanguages(g, opts)

	case "getTopics":
		return getTopics(g, opts)

	case "getLatestSHA":
		// return getRepoBranchSHA(g, opts)
		return getLatestSHA(g, opts)

	}

	return nil, nil, nil
}

func Do(g *Github, entity string, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	if g.Client == nil {
		return nil, nil, errInvalidClient
	}

	/*
		if ExceededRateLimit(g.Client) {
			log.Debugln("new client required as exceeded rate limit detected for the current token, token.old=", g.ctoken, "debug", runtime.WhereAmI())
			g = g.Manager.Fetch()
		}
	*/

	switch entity {
	case "getStars":
		return getStars(g, opts)

	case "getTrends":
		return getTrends(g, opts)

	case "getReleaseTags":
		return getReleaseTags(g, opts)

	case "getCode":
		return getCode(g, opts)

	case "getRepoList":
		return getRepoList(g, opts)

	case "getUser":
		return getUser(g, opts)

	case "getUserOrgs":
		return getUserOrgs(g, opts)

	case "getUserNode":
		return getUserNode(g, opts)

	case "getFollowers":
		return getFollowers(g, opts)

	case "getFollowing":
		return getFollowing(g, opts)

	case "getRepo":
		return getRepo(g, opts)

	case "getReposByOrg":
		return getReposByOrg(g, opts)

	case "getReadme":
		return getReadme(g, opts)

	case "getTree":
		return getTree(g, opts)

	case "getLanguages":
		return getLanguages(g, opts)

	case "getTopics":
		return getTopics(g, opts)

	case "getLatestSHA":
		// return getRepoBranchSHA(g, opts)
		return getLatestSHA(g, opts)

	}

	return nil, nil, nil
}

func nextClient(g *Github, response *github.Response) *Github {
	log.Warnln("new client required, token.old=", g.ctoken, "debug", runtime.WhereAmI())
	go func() {
		g.wg.Add(1)
		defer g.wg.Done()
		g.Reclaim((*response).Reset.Time)
	}()
	return g.Manager.Fetch()
}

func (g *Github) nextClient(response *github.Response) *Github {
	log.Warnln("new client required, token.old=", g.ctoken, "debug", runtime.WhereAmI())
	go func() {
		g.wg.Add(1)
		defer g.wg.Done()
		log.Println("g=", g != nil, "response=", response != nil)
		g.Reclaim((*response).Reset.Time)
	}()

	return g.Manager.Fetch()
}

func getRepoByOrg(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		svc      *Github = g
		user     *github.User
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)

	opts.Runner = "me"
	if opts.Target == nil {
		opts.Target = &Target{}
		opts.Target.Owner = "me"
	}

	goto request

request:
	{

		repos, response, err = svc.Client.Repositories.ListByOrg(
			context.Background(),
			opts.Target.Owner,
			&github.RepositoryListByOrgOptions{
				Type: "public",
				&github.RepositoryListOptions{
					Sort:      "updated",
					Direction: "desc",
					ListOptions: github.ListOptions{
						Page:    opts.Page,
						PerPage: opts.PerPage, // opts.PerPage,
					},
				},
			},
		)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		if repos == nil {
			err = errorMarshallingResponse
			goto finish
		}

		for _, repo := range repos {
			key := fmt.Sprintf("%s/%d/%d", repo.GetFullName(), repo.GetID(), repo.GetStargazersCount())
			mv := mxj.Map(structs.Map(repo))
			if opts.Filter != nil {
				if opts.Filter.Maps != nil {
					res[key] = extractWithMaps(mv, opts.Filter.Maps)
				}
			}
		}

		res["request_url"] = response.Request.URL.String()
		res["object_hash"] = fmt.Sprintf("%x", structhash.Sha1(res, 1))
		goto finish
	}

finish:
	return res, response, nil
}

func getRepoList(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		svc      *Github = g
		repos    []*github.Repository
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)
	goto request

request:
	{
		repos, response, err = svc.Client.Repositories.List(
			context.Background(),
			opts.Target.Owner,
			&github.RepositoryListOptions{
				Sort:      "updated",
				Direction: "desc",
				ListOptions: github.ListOptions{
					Page:    opts.Page,
					PerPage: opts.PerPage,
				},
			},
		)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		if repos == nil {
			err = errorMarshallingResponse
			goto finish
		}

		for _, repo := range repos {
			key := fmt.Sprintf("%s/%d/%d", repo.GetFullName(), repo.GetID(), repo.GetStargazersCount())
			mv := mxj.Map(structs.Map(repo))
			if opts.Filter != nil {
				if opts.Filter.Maps != nil {
					res[key] = extractWithMaps(mv, opts.Filter.Maps)
				}
			}
		}
		goto finish
	}

finish:
	return res, response, nil

}

func getCode(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		cds              = cmap.NewConcurrentMap()
		swg              = sizedwaitgroup.New(64)
		svc      *Github = g
		codes    *github.CodeSearchResult
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)

	goto request

request:
	{
		codes, response, err = svc.Client.Search.Code(
			context.Background(),
			opts.Target.Query,
			&github.SearchOptions{
				Sort:  "indexed",
				Order: "desc",
				ListOptions: github.ListOptions{
					Page:    opts.Page,
					PerPage: opts.PerPage,
				},
			},
		)

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		if err != nil {
			if v := response.Response.Header["Retry-After"]; len(v) > 0 {
				retryAfterSeconds, _ := strconv.ParseInt(v[0], 10, 64) // Error handling is noop.
				retryAfter := time.Duration(retryAfterSeconds) * time.Second
				time.Sleep(retryAfter)
				goto request
			}
			log.Warningln("response.Rate.Reset.Nanosecond()=", response.Rate.Reset.Nanosecond(), "Retry-After=", response.Response.Header.Get("Retry-After"))
			log.Debugln("err=", err)
			goto finish
		}

		if codes == nil {
			err = errorMarshallingResponse
			goto finish
		}

		sliceLength := len(codes.CodeResults)
		log.Println("Running for loop over code repositories matched... len=", sliceLength)
		for i := 0; i < sliceLength; i++ {
			swg.Add()
			go func(i int) {
				defer swg.Done()
				key := fmt.Sprintf("%s/%d/%d", codes.CodeResults[i].Repository.GetFullName(), codes.CodeResults[i].Repository.GetID(), codes.CodeResults[i].Repository.GetStargazersCount())
				// key := fmt.Sprintf("%s/%d", codes.CodeResults[i].GetLogin(), codes.CodeResults[i].GetID())
				mv := mxj.Map(structs.Map(codes.CodeResults[i]))
				if mv == nil {
					return
				}
				row := make(map[string]interface{}, 0)
				if opts.Filter != nil {
					if opts.Filter.Maps != nil {
						row = extractWithMaps(mv, opts.Filter.Maps)
					}
				} else {
					row = mv
				}
				cds.Set(key, row)
			}(i)
		}
		res = cds.GetMapStr()

		swg.Wait()
		log.Println("Finished for loop over code repositories matched...")

		goto finish
	}

finish:
	return res, response, nil

}

func getTrends(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		cds              = cmap.NewConcurrentMap()
		swg              = sizedwaitgroup.New(64)
		svc      *Github = g
		repos    *github.RepositoriesSearchResult
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)

	goto request

request:
	{
		repos, response, err = svc.Client.Search.Repositories(
			context.Background(),
			opts.Target.Query,
			&github.SearchOptions{
				Sort:  "stars",
				Order: "desc",
				ListOptions: github.ListOptions{
					Page:    opts.Page,
					PerPage: opts.PerPage,
				},
			},
		)

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		if err != nil {
			if v := response.Response.Header["Retry-After"]; len(v) > 0 {
				retryAfterSeconds, _ := strconv.ParseInt(v[0], 10, 64) // Error handling is noop.
				retryAfter := time.Duration(retryAfterSeconds) * time.Second
				time.Sleep(retryAfter)
				goto request
			}
			log.Warningln("response.Rate.Reset.Nanosecond()=", response.Rate.Reset.Nanosecond(), "Retry-After=", response.Response.Header.Get("Retry-After"))
			log.Debugln("err=", err.Error())
			goto finish
		}

		if repos == nil {
			err = errorMarshallingResponse
			goto finish
		}

		sliceLength := len(repos.Repositories)
		log.Println("Running for loop over user followed... len=", sliceLength)
		for i := 0; i < sliceLength; i++ {
			swg.Add()
			go func(i int) {
				defer swg.Done()
				key := fmt.Sprintf("%s/%d/%d", repos.Repositories[i].GetFullName(), repos.Repositories[i].GetID(), repos.Repositories[i].GetStargazersCount())
				mv := mxj.Map(structs.Map(repos.Repositories[i]))
				if mv == nil {
					return
				}
				row := make(map[string]interface{}, 0)
				if opts.Filter != nil {
					if opts.Filter.Maps != nil {
						row = extractWithMaps(mv, opts.Filter.Maps)
					}
				} else {
					row = mv
				}
				cds.Set(key, row)
			}(i)
		}
		res = cds.GetMapStr()

		swg.Wait()
		log.Println("Finished for loop over user followed...")

		goto finish
	}

finish:
	// log.Fatal("getTrends...")
	return res, response, nil

}

func getStars(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		// cc       sync.WaitGroup
		svc      *Github = g
		stars    []*github.StarredRepository
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)

	goto request

request:
	{
		stars, response, err = svc.Client.Activity.ListStarred(
			context.Background(),
			opts.Runner,
			&github.ActivityListStarredOptions{
				Sort:      "updated",
				Direction: "desc",
				ListOptions: github.ListOptions{
					Page:    opts.Page,
					PerPage: opts.PerPage,
				},
			},
		)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		if stars == nil {
			err = errorMarshallingResponse
			goto finish
		}

		for _, star := range stars {
			key := fmt.Sprintf("%s/%d/%d", star.Repository.GetFullName(), star.Repository.GetID(), star.Repository.GetStargazersCount())
			mv := mxj.Map(structs.Map(star.Repository))
			if opts.Filter != nil {
				if opts.Filter.Maps != nil {
					res[key] = extractWithMaps(mv, opts.Filter.Maps)
				}
			}
		}
		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, nil

}

func getReleaseTags(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		cds              = cmap.NewConcurrentMap()
		swg              = sizedwaitgroup.New(64)
		svc      *Github = g
		tags     []*github.RepositoryTag
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)

	opts.Runner = "me"
	if opts.Target == nil {
		opts.Target = &Target{}
		opts.Target.Owner = "me"
	}

	goto request

request:
	{
		tags, response, err = svc.Client.Repositories.ListTags(context.Background(), opts.Target.Owner, opts.Target.Name, nil)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		// if response.StatusCode == 403 {
		//	 goto changeClient
		// }

		if tags == nil {
			err = errorMarshallingResponse
			goto finish
		}

		sliceLength := len(tags)
		log.Println("Running for loop over user followed... len=", sliceLength)
		for i := 0; i < sliceLength; i++ {
			swg.Add()
			go func(i int) {
				defer swg.Done()
				key := fmt.Sprintf("%s", tags[i].GetName())
				row := make(map[string]interface{}, 1)
				row[key] = tags
				cds.Set(key, row)
			}(i)
		}
		res = cds.GetMapStr()

		swg.Wait()
		log.Println("Finished for loop over user followed...")

		/*
			for _, tag := range tags {
				key := tag.GetName()
				res[key] = tag
			}
		*/

		// res["object_hash"] = fmt.Sprintf("%x", structhash.Sha1(res, 1))
		goto finish
	}

finish:
	return res, response, nil
}

func getUser(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		svc      *Github = g
		user     *github.User
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)

	opts.Runner = "me"
	if opts.Target == nil {
		opts.Target = &Target{}
		opts.Target.Owner = "me"
	}

	goto request

request:
	{
		user, response, err = svc.Client.Users.Get(context.Background(), opts.Target.Owner)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		// if response.StatusCode == 403 {
		//	 goto changeClient
		// }

		if user == nil {
			err = errorMarshallingResponse
			goto finish
		}

		mv := mxj.Map(structs.Map(user))
		if opts.Filter != nil {
			if opts.Filter.Maps != nil {
				res = extractWithMaps(mv, opts.Filter.Maps)
			}
		}
		res["request_url"] = response.Request.URL.String()
		res["object_hash"] = fmt.Sprintf("%x", structhash.Sha1(res, 1))
		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, nil
}

func getUserOrgs(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		svc      *Github = g
		user     *github.User
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)

	opts.Runner = "me"
	if opts.Target == nil {
		opts.Target = &Target{}
		opts.Target.Owner = "me"
	}

	goto request

request:
	{

		orgs, response, err = svc.Client.Organizations.List(context.Background(), opts.Target.Owner, nil)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		if user == nil {
			err = errorMarshallingResponse
			goto finish
		}

		for _, org := range orgs {
			key := fmt.Sprintf("%s/%d", org.GetLogin(), org.GetID())
			mv := mxj.Map(structs.Map(org))
			if mv == nil {
				continue
			}
			if opts.Filter != nil {
				if opts.Filter.Maps != nil {
					res[key] = extractWithMaps(mv, opts.Filter.Maps)
				}
			} else {
				res[key] = mv
			}
		}

		res["request_url"] = response.Request.URL.String()
		res["object_hash"] = fmt.Sprintf("%x", structhash.Sha1(res, 1))
		goto finish
	}

finish:
	return res, response, nil
}

/*
// list public repositories for org "github"
opt := &github.RepositoryListByOrgOptions{Type: "public"}
repos, _, err := client.Repositories.ListByOrg(context.Background(), "github", opt)
*/

func getFollowers(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		// cds              = cmap.NewConcurrentMap()
		// swg              = sizedwaitgroup.New(64)
		svc      *Github = g
		users    []*github.User
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)
	goto request

request:
	{
		users, response, err = svc.Client.Users.ListFollowers(context.Background(), opts.Target.Owner, &github.ListOptions{Page: opts.Page, PerPage: opts.PerPage})

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		// if response.StatusCode == 403 {
		//	 goto changeClient
		// }

		if users == nil {
			err = errorMarshallingResponse
			goto finish
		}

		for _, user := range users {
			key := fmt.Sprintf("%s/%d", user.GetLogin(), user.GetID())
			mv := mxj.Map(structs.Map(user))
			if mv == nil {
				continue
			}
			if opts.Filter != nil {
				if opts.Filter.Maps != nil {
					res[key] = extractWithMaps(mv, opts.Filter.Maps)
				}
			} else {
				res[key] = mv
			}
		}

		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, err
}

func getFollowing(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		cds              = cmap.NewConcurrentMap()
		swg              = sizedwaitgroup.New(64)
		svc      *Github = g
		users    []*github.User
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)
	goto request

request:
	{
		users, response, err = svc.Client.Users.ListFollowing(context.Background(), opts.Target.Owner, &github.ListOptions{Page: opts.Page, PerPage: opts.PerPage})

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		// if response.StatusCode == 403 {
		// 	 goto changeClient
		// }

		if users == nil {
			err = errorMarshallingResponse
			goto finish
		}

		sliceLength := len(users)
		log.Println("Running for loop over user followed... len=", sliceLength)
		for i := 0; i < sliceLength; i++ {
			swg.Add()
			go func(i int) {
				defer swg.Done()
				key := fmt.Sprintf("%s/%d", users[i].GetLogin(), users[i].GetID())
				mv := mxj.Map(structs.Map(users[i]))
				if mv == nil {
					return
				}
				row := make(map[string]interface{}, 0)
				if opts.Filter != nil {
					if opts.Filter.Maps != nil {
						row = extractWithMaps(mv, opts.Filter.Maps)
					}
				} else {
					row = mv
				}
				cds.Set(key, row)
			}(i)
		}
		res = cds.GetMapStr()

		swg.Wait()
		log.Println("Finished for loop over user followed...")

		/*
			for _, user := range users {
				key := fmt.Sprintf("%s/%d", user.GetLogin(), user.GetID())
				mv := mxj.Map(structs.Map(user))
				if mv == nil {
					continue
				}

				if opts.Filter != nil {
					if opts.Filter.Maps != nil {
						res[key] = extractWithMaps(mv, opts.Filter.Maps)
					}
				} else {
					res[key] = mv
				}
			}
		*/

		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, err
}

func getRepo(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		svc      *Github = g
		repo     *github.Repository
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)
	goto request

request:
	{
		repo, response, err = svc.Client.Repositories.Get(context.Background(), opts.Target.Owner, opts.Target.Name)
		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		if repo == nil {
			err = errorMarshallingResponse
			goto finish
		}

		mv := mxj.Map(structs.Map(repo))
		if opts.Filter != nil {
			if opts.Filter.Maps != nil {
				res = extractWithMaps(mv, opts.Filter.Maps)
			}
		}
		res["request_url"] = response.Request.URL.String()
		res["object_hash"] = fmt.Sprintf("%x", structhash.Sha1(res, 1))
		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, err
}

func getTopics(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		cds              = cmap.NewConcurrentMap()
		swg              = sizedwaitgroup.New(64)
		svc      *Github = g
		topics   []string
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)
	goto request

request:
	{
		topics, response, err = svc.Client.Repositories.ListAllTopics(context.Background(), opts.Target.Owner, opts.Target.Name)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		if topics == nil {
			err = errorMarshallingResponse
			goto finish
		}

		sliceLength := len(topics)
		log.Println("Running for loop over repo topics referenced... len=", sliceLength)
		for i := 0; i < sliceLength; i++ {
			swg.Add()
			go func(i int) {
				defer swg.Done()

				key := fmt.Sprintf("%s", topics[i])
				row := make(map[string]interface{}, 0)
				row["label"] = topics[i]
				row["owner"] = opts.Target.Owner
				row["name"] = opts.Target.Name
				row["remote_id"] = strconv.Itoa(opts.Target.RepoId)
				row["request_url"] = response.Request.URL.String()
				cds.Set(key, row)
			}(i)

		}
		res = cds.GetMapStr()

		swg.Wait()
		log.Println("Finished for loop over repo topics referenced...")

		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, nil
}

func getLanguages(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		cds              = cmap.NewConcurrentMap()
		swg              = sizedwaitgroup.New(64)
		svc      *Github = g
		langs    map[string]int
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)
	goto request

request:
	{
		langs, response, err = svc.Client.Repositories.ListLanguages(context.Background(), opts.Target.Owner, opts.Target.Name)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		if langs == nil {
			err = errorMarshallingResponse
			goto finish
		}

		sliceLength := len(langs)
		log.Println("Running for loop over repo languages referenced... len=", sliceLength)
		for lang, _ := range langs {
			swg.Add()
			go func(lang string) {
				defer swg.Done()
				key := fmt.Sprintf("%s", lang)
				row := make(map[string]interface{}, 0)
				row["lang"] = lang
				row["owner"] = opts.Target.Owner
				row["name"] = opts.Target.Name
				row["remote_id"] = strconv.Itoa(opts.Target.RepoId)
				row["request_url"] = response.Request.URL.String()
				cds.Set(key, row)
			}(lang)
		}
		res = cds.GetMapStr()

		swg.Wait()
		log.Println("Finished for loop over repo languages referenced...")

		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, nil
}

func getRepoBranchSHA(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		svc      *Github = g
		ref      *github.Branch
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)
	goto request

request:
	{
		ref, response, err = svc.Client.Repositories.GetBranch(context.Background(), opts.Target.Owner, opts.Target.Name, opts.Target.Branch)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		if ref.Commit == nil {
			err = errorMarshallingResponse
			goto finish
		}

		res["sha"] = *ref.Commit.SHA
		res["owner"] = opts.Target.Owner
		res["name"] = opts.Target.Name
		res["branch"] = opts.Target.Branch
		res["remote_repo_id"] = opts.Target.RepoId
		res["request_url"] = response.Request.URL.String()
		res["object_hash"] = fmt.Sprintf("%x", structhash.Sha1(res, 1))
		pp.Println("ref=", ref, "res=", res)

		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, nil
}

func getLatestSHA(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		svc      *Github = g
		ref      *github.Reference
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)
	goto request

request:
	{
		if opts.Target.Branch == "" {
			opts.Target.Branch = "master"
		}

		ref, response, err = svc.Client.Git.GetRef(context.Background(), opts.Target.Owner, opts.Target.Name, "refs/heads/"+opts.Target.Branch)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		// if response.StatusCode == 403 {
		//	 goto changeClient
		// }

		if ref == nil {
			err = errorMarshallingResponse
			goto finish
		}

		res["sha"] = *ref.Object.SHA
		res["owner"] = opts.Target.Owner
		res["name"] = opts.Target.Name
		res["branch"] = opts.Target.Branch
		res["remote_repo_id"] = opts.Target.RepoId
		res["request_url"] = response.Request.URL.String()
		res["object_hash"] = fmt.Sprintf("%x", structhash.Sha1(res, 1))
		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, nil
}

func getTree(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		svc      *Github = g
		tree     *github.Tree
		response *github.Response
		cds      = cmap.NewConcurrentMap()
		swg      = sizedwaitgroup.New(64)
		res      = make(map[string]interface{}, 0)
		err      error
	)
	goto request

request:
	{
		tree, response, err = svc.Client.Git.GetTree(context.Background(), opts.Target.Owner, opts.Target.Name, opts.Target.Ref, true)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		//if response.StatusCode == 403 {
		//	goto changeClient
		//}

		if tree == nil {
			err = errorMarshallingResponse
			goto finish
		}

		sliceLength := len(tree.Entries)
		log.Println("Running for loop over the git tree... len=", sliceLength)
		for i := 0; i < sliceLength; i++ {
			swg.Add()
			go func(i int) {
				defer swg.Done()
				val := tree.Entries[i]
				row := make(map[string]interface{}, 0)
				entry_path := val.GetPath()
				row["path"] = entry_path
				row["owner"] = opts.Target.Owner
				row["name"] = opts.Target.Name
				row["remote_id"] = strconv.Itoa(opts.Target.RepoId)
				row["request_url"] = response.Request.URL.String()
				row["size"] = val.GetSize()
				// row["sha"] = val.GetSHA()
				// row["url"] = val.GetURL()
				key := fmt.Sprintf("entry-%d", i)
				cds.Set(key, row)
				log.Infoln("key=", key, "i=", i)
			}(i)

		}
		res = cds.GetMapStr()

		swg.Wait()
		log.Println("Finished for loop over the git tree...")
		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, nil
}

func getReadme(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var (
		svc      *Github = g
		readme   *github.RepositoryContent
		response *github.Response
		res      = make(map[string]interface{}, 0)
		err      error
	)
	goto request

request:
	{
		readme, response, err = svc.Client.Repositories.GetReadme(context.Background(), opts.Target.Owner, opts.Target.Name, nil)

		if err != nil {
			goto finish
		}

		if response == nil {
			err = errorResponseIsNull
			goto finish
		}

		//if response.StatusCode == 403 {
		//	goto changeClient
		//}

		if readme == nil {
			err = errorMarshallingResponse
			goto finish
		}

		content, _ := readme.GetContent()
		readme.Content = &content

		mv := mxj.Map(structs.Map(readme))
		if opts.Filter != nil {
			if opts.Filter.Maps != nil {
				res = extractWithMaps(mv, opts.Filter.Maps)
			}
		}

		res["owner"] = opts.Target.Owner
		res["name"] = opts.Target.Name
		res["remote_repo_id"] = opts.Target.RepoId
		res["request_url"] = response.Request.URL.String()
		res["object_hash"] = fmt.Sprintf("%x", structhash.Sha1(res, 1))
		goto finish
	}

	/*
	   changeClient:
	   	{
	   		if ok, resetAt := ExceededRateLimit(svc.Client); ok {
	   			log.Errorln("resetAt: ", resetAt)
	   			go func() {
	   				cc.Add(1)
	   				defer cc.Done()

	   				Reclaim(svc, resetAt)
	   			}()
	   			svc = clientManager.Fetch()
	   		}
	   		goto request
	   	}
	*/

finish:
	return res, response, nil
}

// Run starts the dispatcher and pushes a new request for the root user onto
// the queue. Returns the *UserNode that is received on the done channel.
func getUserNode(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) { // *UserNode { // start, end string, opts.Workers int, g *Github) *UserNode {
	// defaultCLI, ctx = g.client, context.Background()

	if opts.Workers <= 0 {
		opts.Workers = 6
	}

	startDispatcher(opts.Workers)
	origin, target = opts.Start, opts.End
	jobQueue <- jobRequest{User: newUserNode(origin, nil)}

	for {
		select {
		case user := <-done:
			// pp.Println(user)
			return mapstruct.Struct2Map(user), nil, nil
		}
	}

	//finish:
	//	return res, response, nil
}

func isNotFound(err error) bool {
	errResp, ok := errorsp.Cause(err).(*github.ErrorResponse)
	if !ok {
		return false
	}
	return errResp.Response.StatusCode == http.StatusNotFound
}

// verifyRepo checks all essential fields of a Repository structure for nil
// values. An error is returned if one of the essential field is nil.
func verifyRepo(repo *github.Repository) error {
	if repo == nil {
		return newInvalidStructError("verifyRepo: repo is nil")
	}

	var err *invalidStructError
	if repo.ID == nil {
		err = newInvalidStructError("verifyRepo: contains nil fields:").AddField("ID")
	} else {
		err = newInvalidStructError(fmt.Sprintf("verifyRepo: repo #%d contains nil fields: ", *repo.ID))
	}

	if repo.Name == nil {
		err.AddField("Name")
	}

	if repo.Language == nil {
		err.AddField("Language")
	}

	if repo.CloneURL == nil {
		err.AddField("CloneURL")
	}

	if repo.Owner == nil {
		err.AddField("Owner")
	} else {
		if repo.Owner.Login == nil {
			err.AddField("Owner.Login")
		}
	}

	if repo.Fork == nil {
		err.AddField("Fork")
	}

	if err.FieldsLen() > 0 {
		return err
	}

	return nil
}

func extractWithMaps(mv mxj.Map, fields map[string]string) map[string]interface{} {
	l := make(map[string]interface{}, len(fields))
	for key, path := range fields {
		var node []interface{}
		var merr error
		node, merr = mv.ValuesForPath(path)
		if merr != nil {
			log.Fatalln("[extractWithMaps] Error: ", merr, "key=", key, "path=", path)
			continue
		}

		/*
			switch length := len(node); {
			case length == 1:
				l[key] = node[0]
			case length > 1:
				l[key] = node
			default:
				continue
			}
		*/
		if len(node) > 1 {
			l[key] = node
		} else if len(node) == 1 {
			l[key] = node[0]
		}
	}
	return l
}

func extractFlatten(mv mxj.Map, fields []string) map[string]interface{} {
	l := make(map[string]interface{}, len(fields))
	for _, path := range fields {
		// var node []interface{}
		// var merr error
		node, _ := mv.ValuesForPath(path)
		// if merr != nil {
		//	log.Fatalln("Error: ", merr)
		// }
		if node != nil {
			log.Warnln("node len=", len(node))
			if len(node) > 2 {
				l[path] = node
			} else {
				l[path] = node[0]
			}
		}
	}
	return l
}

func extractBlocks(mv mxj.Map, items string, fields map[string][]string) map[string]interface{} {
	l := make(map[string]interface{}, len(fields))
	for attr, field := range fields {
		var keyPath string
		var node []interface{}
		if len(field) == 1 {
			keyPath = fmt.Sprintf("%#s", field[0])
			node, _ = mv.ValuesForPath(keyPath)
			// log.Debugln("attr=", attr, "keyPath=", keyPath, "node=", node)
		} else {
			w := make(map[string]interface{}, len(field))
			var merr error
			for _, whl := range field {
				keyParts := strings.Split(whl, ".")
				keyName := keyParts[len(keyParts)-1]
				keyPath = fmt.Sprintf("%#s", whl)
				// log.Debugln("attr=", attr, "keyPath=", keyPath, "keyName=", keyName)
				node, merr = mv.ValuesForPath(keyPath)
				if merr != nil {
					log.Fatalln("[extractBlocks] Error: ", merr)
				}
				if node != nil {
					if len(node) == 1 {
						w[keyName] = node[0]
					} else if len(node) > 1 {
						w[keyName] = node
					}
				}
			}
			l[attr] = w
			continue
		}
		if len(node) == 1 {
			l[attr] = node[0]
			// log.Debugln("attr=", attr, "node[0]=", node[0])
		} else if len(node) > 1 {
			// log.Debugln("attr=", attr, "node=", node)
			l[attr] = node
		}
	}
	// log.Println(l)
	return l
}
