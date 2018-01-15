package github

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/src-d/enry/data"

	"github.com/sniperkit/xtask/util/runtime"
	"github.com/sniperkit/xutil/plugin/format/convert/mxj/pkg"
	"github.com/sniperkit/xutil/plugin/struct"
)

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

	res := make(map[string]interface{}, 0)
	for _, star := range stars {
		key := fmt.Sprintf("%s/%d", star.Repository.GetFullName(), star.Repository.GetID())
		mv := mxj.Map(structs.Map(star.Repository))
		if opts.Filter != nil {
			if opts.Filter.Maps != nil {
				res[key] = extractWithMaps(mv, opts.Filter.Maps)
			}
		}
	}

	return res, response, nil
}

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

	res := make(map[string]interface{}, 0)
	mv := mxj.Map(structs.Map(user))
	if opts.Filter != nil {
		if opts.Filter.Maps != nil {
			res = extractWithMaps(mv, opts.Filter.Maps)
		}
	}

	return res, response, nil
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

	res := make(map[string]interface{}, 0)
	mv := mxj.Map(structs.Map(repo))
	if opts.Filter != nil {
		if opts.Filter.Maps != nil {
			res = extractWithMaps(mv, opts.Filter.Maps)
		}
	}

	return res, response, nil
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

	res := make(map[string]interface{}, len(topics))
	for _, topic := range topics {
		key := fmt.Sprintf("topic-%d", topic)
		row := make(map[string]interface{}, 4)
		row["topic"] = topic
		row["owner"] = opts.Target.Owner
		row["name"] = opts.Target.Name
		row["remote_id"] = strconv.Itoa(opts.Target.RepoId)
		res[key] = row
	}

	return res, response, nil
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
	fm["owner"] = opts.Target.Owner
	fm["name"] = opts.Target.Name
	fm["branch"] = opts.Target.Branch
	fm["remote_repo_id"] = opts.Target.RepoId

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

	// filters := []string{"CMakeLists.txt", "Dockerfile", "docker-compose", "crane.yaml", "crane.yml", ""}

	res := make(map[string]interface{}, 0)
	for k, entry := range tree.Entries {
		row := make(map[string]interface{}, 7)
		if !filterTree(entry.GetPath()) {
			continue
		}
		row["path"] = entry.GetPath()
		row["owner"] = opts.Target.Owner
		row["name"] = opts.Target.Name
		row["remote_id"] = strconv.Itoa(opts.Target.RepoId)
		// row["sha"] = entry.GetSHA()
		// row["size"] = entry.GetSize()
		// row["url"] = entry.GetURL()
		key := fmt.Sprintf("entry-%d", k)
		res[key] = row
	}

	return res, response, nil
}

var typeListFiles = map[string][]string{
	"cmake":    []string{"CMakeLists.txt"},
	"docker":   []string{"Dockerfile"},
	"crystal":  []string{"Projectfile"},
	"markdown": []string{".markdown", ".md", ".mdown", ".mkdn"},
	"asciidoc": []string{".adoc", ".asc", ".asciidoc"},
	"groovy":   []string{".groovy", ".gradle"},
	"msbuild":  []string{".csproj", ".fsproj", ".vcxproj", ".proj", ".props", ".targets"},
	"wiki":     []string{".mediawiki", ".wiki"},
	"make":     []string{"gnumakefile", "Gnumakefile", "makefile", "Makefile"},
	"mk":       []string{"mkfile"},
	"ruby":     []string{"Gemfile", ".irbrc", "Rakefile"},
	"toml":     []string{"Cargo.lock"},
	"zsh":      []string{"zshenv", ".zshenv", "zprofile", ".zprofile", "zshrc", ".zshrc", "zlogin", ".zlogin", "zlogout", ".zlogout"},
}

var lang_path_map = map[string]string{
	`rakefile`: `ruby`,
	`/(Makefile|CMakeLists.txt|Imakefile|makepp|configure)$`: `make`,
	`/config$`:     `conf`,
	`/zsh/_[^/]+$`: `sh`,
	`patch`:        `diff`,
}

func filterTree(filepath string) bool {
	// data.DocumentationMatchers
	// data.LanguagesByFilename
	// data.LanguagesByInterpreter
	return false
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

	res := make(map[string]interface{}, len(opts.Filter.Maps))
	mv := mxj.Map(structs.Map(readme))
	if opts.Filter != nil {
		if opts.Filter.Maps != nil {
			res = extractWithMaps(mv, opts.Filter.Maps)
		}
	}

	res["owner"] = opts.Target.Owner
	res["name"] = opts.Target.Name
	res["remote_repo_id"] = opts.Target.RepoId

	return res, response, nil
}

func extractWithMaps(mv mxj.Map, fields map[string]string) map[string]interface{} {
	l := make(map[string]interface{}, len(fields))
	for key, path := range fields {
		var node []interface{}
		var merr error
		node, merr = mv.ValuesForPath(path)
		if merr != nil {
			log.Warnln("Error: ", merr)
			continue
		}
		if len(node) > 1 {
			l[key] = node
		} else {
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
					log.Fatalln("Error: ", merr)
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
