package github

import (
	"context"
	"regexp"
	"time"

	"github.com/google/go-github/github"

	"github.com/sniperkit/xtask/util/runtime"

	"github.com/sniperkit/xutil/plugin/format/convert/mxj"
	"github.com/sniperkit/xutil/plugin/format/json/flatten"
	"github.com/sniperkit/xutil/plugin/struct"
	// "github.com/k0kubun/pp"
)

/*
	// Once again, get the same repository, but override the request so we bypass the cache and hit GitHub's API.
	reqModifyingTransport.Override(regexp.MustCompile(`^/repos/sourcegraph/apiproxy$`), apiproxy.NoCache, true)
*/

func (g *Github) counterTrack(name string, incr int) {
	go func() {
		g.counters.Increment(name, incr)
	}()
}

func (g *Github) GetFunc(entity string, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	if g.client == nil {
		g.client = getClient(g.ctoken)
	}
	switch entity {
	case "getStars":
		return getStars(g, opts)
	case "getUser":
		return getUser(g, opts)
	case "getRepo":
		return getRepo(g, opts)
	case "getReadme":
		return getReadme(g, opts)
	case "getTree":
		return getTree(g, opts)
	case "getTopics":
		return getTopics(g, opts)
	case "getLatestSHA":
		return getLatestSHA(g, opts)
	}
	return nil, nil, nil
}

func getStars(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var stars []*github.StarredRepository
	var response *github.Response
	get := func() error {
		var err error
		stars, response, err = g.client.Activity.ListStarred(
			context.Background(),
			opts.Runner,
			&github.ActivityListStarredOptions{
				Sort:      "updated",
				Direction: "desc", // desc
				ListOptions: github.ListOptions{
					Page:    opts.Page,
					PerPage: opts.PerPage,
				},
			},
		)
		if response == nil {
			return err
		}
		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				log.Warnln("new client required, debug=", runtime.WhereAmI())
				oldToken := g.ctoken
				newToken := g.getNextToken(oldToken)
				newClient := g.getClient(newToken)
				g.client = newClient
			}
			return err
		}
		return nil
	}
	if err := retryRegistrationFunc(get); err != nil {
		log.Errorln("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	if response == nil {
		return nil, nil, errorResponseIsNull
	}
	if stars == nil {
		return nil, response, errorMarshallingResponse
	}

	// Check if server is a struct or a pointer to struct
	/*
		if !structs.IsStruct(stars) {
			pp.Println(stars)
			return nil, response, errorNotStruct
		}

		sm := structs.Map(stars)
	*/

	var items []interface{}
	for _, star := range stars {
		items = append(items, structs.Map(star))
	}

	var mv mxj.Map
	mxj.JsonUseNumber = true
	mv = mxj.Map(map[string]interface{}{"items": items})

	mxj.LeafUseDotNotation()
	leafPaths := leafPathsPatterns(mv.LeafPaths())
	log.Println("mv.LeafPaths(): ")
	for _, p := range leafPaths {
		log.Println("path:", p)
	}

	log.Fatal("test\n")

	res := make(map[string]interface{}, len(stars))
	for _, star := range stars {
		key := star.Repository.GetFullName()
		res[key] = star.GetStarredAt()
	}
	return res, response, nil
}

func leafPathsPatterns(input []string) []string {
	var output []string
	var re = regexp.MustCompile(`.([0-9]+)`)
	for _, value := range input {
		value = re.ReplaceAllString(value, `[*]`)
		if !contains(output, value) {
			output = append(output, value)
		}
	}
	return dedup(output)
}

func contains(input []string, match string) bool {
	for _, value := range input {
		if value == match {
			return true
		}
	}
	return false
}

func dedup(input []string) []string {
	var output []string
	for _, value := range input {
		if !contains(output, value) {
			output = append(output, value)
		}
	}
	return output
}

/*
	ponzuClient := &http.Client{}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	starStruct := structs.New(s)
	for _, f := range starStruct.Fields() {
		if f.IsEmbedded() {
			continue
		}
		if f.Name() == "Tags" {
			for i, v := range s.Tags {
				writer.WriteField(fmt.Sprintf("tags.%d", i), v)
			}
			continue
		}
		if f.IsZero() == false {
			writer.WriteField(f.Tag("json"), fmt.Sprint(f.Value()))
		}
	}
*/

func getUser(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var user *github.User
	var response *github.Response
	get := func() error {
		var err error
		user, response, err = g.client.Users.Get(context.Background(), opts.Target.Owner)
		if response == nil {
			return err
		}
		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				log.Warnln("new client required, debug=", runtime.WhereAmI())
				oldToken := g.ctoken
				newToken := g.getNextToken(oldToken)
				g.client = g.getClient(newToken)
			}
			return err
		}
		return nil
	}
	if err := retryRegistrationFunc(get); err != nil {
		log.Errorln("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	if response == nil {
		return nil, nil, errorResponseIsNull
	}
	if user == nil {
		return nil, response, errorMarshallingResponse
	}
	var fm map[string]interface{}
	var err error
	m := structs.Map(user)
	fm, err = flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		log.Errorln("error: ", err.Error(), "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	return fm, response, nil
}

func getRepo(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var repo *github.Repository
	var response *github.Response
	get := func() error {
		var err error
		repo, response, err = g.client.Repositories.Get(context.Background(), opts.Target.Owner, opts.Target.Name)
		if response == nil {
			return err
		}
		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				log.Warnln("new client required, debug=", runtime.WhereAmI())
				oldToken := g.ctoken
				newToken := g.getNextToken(oldToken)
				g.client = g.getClient(newToken)
			}
			return err
		}
		return nil
	}
	if err := retryRegistrationFunc(get); err != nil {
		log.Errorln("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	if response == nil {
		return nil, nil, errorResponseIsNull
	}
	if repo == nil {
		return nil, response, errorMarshallingResponse
	}
	var fm map[string]interface{}
	var err error
	m := structs.Map(repo)
	fm, err = flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		log.Errorln("error: ", err.Error(), "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	return fm, response, nil
}

func getTopics(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var topics []string
	var response *github.Response
	get := func() error {
		var err error
		topics, response, err = g.client.Repositories.ListAllTopics(context.Background(), opts.Target.Owner, opts.Target.Name)
		if response == nil {
			return err
		}
		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				oldToken := g.ctoken
				newToken := g.getNextToken(oldToken)
				log.Warnln("new client required, oldToken=", oldToken, ", newToken:", newToken, " debug=", runtime.WhereAmI())
				g.client = g.getClient(newToken)
			}
			return err
		}
		return nil
	}
	if err := retryRegistrationFunc(get); err != nil {
		log.Errorln("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	if response == nil {
		return nil, nil, errorResponseIsNull
	}
	if topics == nil {
		return nil, response, errorMarshallingResponse
	}
	fm := make(map[string]interface{}, 1)
	fm["topics"] = topics
	return fm, response, nil
}

func getLatestSHA(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var ref *github.Reference
	var response *github.Response
	get := func() error {
		var err error
		if opts.Target.Branch == "" {
			opts.Target.Branch = "master"
		}
		ref, response, err = g.client.Git.GetRef(context.Background(), opts.Target.Owner, opts.Target.Name, "refs/heads/"+opts.Target.Branch)
		if response == nil {
			return err
		}
		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				oldToken := g.ctoken
				newToken := g.getNextToken(oldToken)
				log.Warnln("new client required, oldToken=", oldToken, ", newToken:", newToken, " debug=", runtime.WhereAmI())
				g.client = g.getClient(newToken)
			}
			return err
		}
		return nil
	}
	if err := retryRegistrationFunc(get); err != nil {
		log.Errorln("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	if response == nil {
		return nil, nil, errorResponseIsNull
	}
	if ref == nil {
		return nil, response, errorMarshallingResponse
	}
	fm := make(map[string]interface{}, 1)
	fm["sha"] = *ref.Object.SHA
	return fm, response, nil
}

func getTree(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var tree *github.Tree
	var response *github.Response
	get := func() error {
		var err error
		tree, response, err = g.client.Git.GetTree(context.Background(), opts.Target.Owner, opts.Target.Name, opts.Target.Ref, true)
		if response == nil {
			return err
		}
		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				oldToken := g.ctoken
				newToken := g.getNextToken(oldToken)
				log.Warnln("new client required, oldToken=", oldToken, ", newToken:", newToken, " debug=", runtime.WhereAmI())
				g.client = g.getClient(newToken)
			}
			return err
		}
		return nil
	}
	if err := retryRegistrationFunc(get); err != nil {
		log.Errorln("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	if response == nil {
		return nil, nil, errorResponseIsNull
	}
	if tree == nil {
		return nil, response, errorMarshallingResponse
	}
	var fm map[string]interface{}
	var err error
	m := structs.Map(tree)
	fm, err = flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		log.Errorln("error: ", err.Error(), "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	return fm, response, nil
}

func getReadme(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	defer funcTrack(time.Now())

	var readme *github.RepositoryContent
	var response *github.Response
	get := func() error {
		var err error
		readme, response, err = g.client.Repositories.GetReadme(context.Background(), opts.Target.Owner, opts.Target.Name, nil)
		if response == nil {
			return err
		}
		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				oldToken := g.ctoken
				newToken := g.getNextToken(oldToken)
				log.Warnln("new client required, resp.StatusCode=", response.StatusCode, ", resp.Rate=", response.Rate, ", oldToken=", oldToken, ", newToken:", newToken, " debug=", runtime.WhereAmI())
				g.client = g.getClient(newToken)
			}
			return err
		}
		return nil
	}
	if err := retryRegistrationFunc(get); err != nil {
		log.Errorln("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	if response == nil {
		return nil, nil, errorResponseIsNull
	}
	if readme == nil {
		return nil, response, errorMarshallingResponse
	}
	content, _ := readme.GetContent()
	readme.Content = &content
	var fm map[string]interface{}
	var err error
	m := structs.Map(readme)
	fm, err = flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		log.Errorln("error: ", err.Error(), "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	return fm, response, err
}
