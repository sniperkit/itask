package github

func Init() {
	defaultOpts = &Options{}
	defaultOpts.Page = 1
	defaultOpts.PerPage = 100
	Service = New(nil, defaultOpts)
}
