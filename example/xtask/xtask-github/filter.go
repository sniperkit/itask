package main

import (
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
)

var (
	Numeric              = `^(\d+)$`
	AlphaNumeric         = `^([0-9A-Za-z]+)$`
	Alpha                = `^([A-Za-z]+)$`
	AlphaCapsOnly        = `^([A-Z]+)$`
	AlphaNumericCapsOnly = `^([0-9A-Z]+)$`
	Url                  = `^((http?|https?|ftps?):\/\/)?([\da-z\.-]+)\.([a-z\.]{2,6})([\/\w \.-]*)*\/?$`
	Email                = `^(.+@([\da-z\.-]+)\.([a-z\.]{2,6}))$`
	HashtagHex           = `^#([a-f0-9]{6}|[a-f0-9]{3})$`
	ZeroXHex             = `^0x([a-f0-9]+|[A-F0-9]+)$`
	IPv4                 = `^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
	IPv6                 = `^([0-9A-Fa-f]{0,4}:){2,7}([0-9A-Fa-f]{1,4}$|((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.|$)){4})$`
	whitePatternsList    = []string{"CMakeLists.txt", "Dockerfile", "docker-compose", "crane.yaml", "crane.yml"}
	ignorePatternsList   = []string{`^Godeps`, `^Godeps/Godeps\.json`, `^vendor/manifest`, `^vendor`}
	docPatternsList      = []string{"md", "markdown", "mdown", "mkdn", "mdwn", "mdtxt", "txt", "text", "doc", "htm", "html"}
	depPatternsList      = []string{`(^|/)cache/`, `^[Dd]ependencies/`, `^deps/`, `^tools/`, `(^|/)configure$`, `(^|/)configure.ac$`, `(^|/)config.guess$`, `(^|/)config.sub$`, `cpplint.py`, `node_modules/`, `bower_components/`, `^rebar$`, `erlang.mk`, `Godeps/_workspace/`, `(\.|-)min\.(js|css)$`, `([^\s]*)import\.(css|less|scss|styl)$`, `(^|/)bootstrap([^.]*)\.(js|css|less|scss|styl)$`, `(^|/)custom\.bootstrap([^\s]*)(js|css|less|scss|styl)$`, `(^|/)font-awesome\.(css|less|scss|styl)$`, `(^|/)foundation\.(css|less|scss|styl)$`, `(^|/)normalize\.(css|less|scss|styl)$`, `(^|/)[Bb]ourbon/.*\.(css|less|scss|styl)$`, `(^|/)animate\.(css|less|scss|styl)$`, `third[-_]?party/`, `3rd[-_]?party/`, `vendors?/`, `extern(al)?/`, `(^|/)[Vv]+endor/`, `^debian/`, `run.n$`, `bootstrap-datepicker/`, `(^|/)jquery([^.]*)\.js$`, `(^|/)jquery\-\d\.\d+(\.\d+)?\.js$`, `(^|/)jquery\-ui(\-\d\.\d+(\.\d+)?)?(\.\w+)?\.(js|css)$`, `(^|/)jquery\.(ui|effects)\.([^.]*)\.(js|css)$`, `jquery.fn.gantt.js`, `jquery.fancybox.(js|css)`, `fuelux.js`, `(^|/)jquery\.fileupload(-\w+)?\.js$`, `(^|/)slick\.\w+.js$`, `(^|/)Leaflet\.Coordinates-\d+\.\d+\.\d+\.src\.js$`, `leaflet.draw-src.js`, `leaflet.draw.css`, `Control.FullScreen.css`, `Control.FullScreen.js`, `leaflet.spin.js`, `wicket-leaflet.js`, `.sublime-project`, `.sublime-workspace`, `(^|/)prototype(.*)\.js$`, `(^|/)effects\.js$`, `(^|/)controls\.js$`, `(^|/)dragdrop\.js$`, `(.*?)\.d\.ts$`, `(^|/)mootools([^.]*)\d+\.\d+.\d+([^.]*)\.js$`, `(^|/)dojo\.js$`, `(^|/)MochiKit\.js$`, `(^|/)yahoo-([^.]*)\.js$`, `(^|/)yui([^.]*)\.js$`, `(^|/)ckeditor\.js$`, `(^|/)tiny_mce([^.]*)\.js$`, `(^|/)tiny_mce/(langs|plugins|themes|utils)`, `(^|/)MathJax/`, `(^|/)Chart\.js$`, `(^|/)[Cc]ode[Mm]irror/(\d+\.\d+/)?(lib|mode|theme|addon|keymap|demo)`, `(^|/)shBrush([^.]*)\.js$`, `(^|/)shCore\.js$`, `(^|/)shLegacy\.js$`, `(^|/)angular([^.]*)\.js$`, `(^|\/)d3(\.v\d+)?([^.]*)\.js$`, `(^|/)react(-[^.]*)?\.js$`, `(^|/)modernizr\-\d\.\d+(\.\d+)?\.js$`, `(^|/)modernizr\.custom\.\d+\.js$`, `(^|/)knockout-(\d+\.){3}(debug\.)?js$`, `(^|/)docs?/_?(build|themes?|templates?|static)/`, `(^|/)admin_media/`, `^fabfile\.py$`, `^waf$`, `^.osx$`, `\.xctemplate/`, `\.imageset/`, `^Carthage/`, `^Pods/`, `(^|/)Sparkle/`, `Crashlytics.framework/`, `Fabric.framework/`, `gitattributes$`, `gitignore$`, `gitmodules$`, `(^|/)gradlew$`, `(^|/)gradlew\.bat$`, `(^|/)gradle/wrapper/`, `-vsdoc\.js$`, `\.intellisense\.js$`, `(^|/)jquery([^.]*)\.validate(\.unobtrusive)?\.js$`, `(^|/)jquery([^.]*)\.unobtrusive\-ajax\.js$`, `(^|/)[Mm]icrosoft([Mm]vc)?([Aa]jax|[Vv]alidation)(\.debug)?\.js$`, `^[Pp]ackages\/.+\.\d+\/`, `(^|/)extjs/.*?\.js$`, `(^|/)extjs/.*?\.xml$`, `(^|/)extjs/.*?\.txt$`, `(^|/)extjs/.*?\.html$`, `(^|/)extjs/.*?\.properties$`, `(^|/)extjs/.sencha/`, `(^|/)extjs/docs/`, `(^|/)extjs/builds/`, `(^|/)extjs/cmd/`, `(^|/)extjs/examples/`, `(^|/)extjs/locale/`, `(^|/)extjs/packages/`, `(^|/)extjs/plugins/`, `(^|/)extjs/resources/`, `(^|/)extjs/src/`, `(^|/)extjs/welcome/`, `(^|/)html5shiv\.js$`, `^[Tt]ests?/fixtures/`, `^[Ss]pecs?/fixtures/`, `(^|/)cordova([^.]*)\.js$`, `(^|/)cordova\-\d\.\d(\.\d)?\.js$`, `foundation(\..*)?\.js$`, `^Vagrantfile$`, `.[Dd][Ss]_[Ss]tore$`, `^vignettes/`, `^inst/extdata/`, `octicons.css`, `sprockets-octicons.scss`, `(^|/)activator$`, `(^|/)activator\.bat$`, `proguard.pro`, `proguard-rules.pro`, `^puphpet/`, `(^|/)\.google_apis/`}
)

var wordFiltersMap = map[string]*github.FilterInfo{
	`ignore`: &github.FilterInfo{
		Ignore:  true,
		Extract: false,
		Regexp:  []string{`[Ll]ibrary/`, `ExportedObj/`, `/.vs/`, `[Tt]emp/`, `[Oo]bj/`, `[Bb]uild/`, `[Bb]uilds/`, `^Carthage/`, `^Pods/`, `.[Dd][Ss]_[Ss]tore$`, `Generated\ Files/`, `ipch/`, `.sublime-project`, `.sublime-workspace`, `third[-_]?party/`, `3rd[-_]?party/`, `vendors?/`, `extern(al)?/`, `(^|/)[Vv]+endor/`, `Godeps`, `(^|/)vendor/`, `node_modules`, `CMakeFiles`, `CMakeScripts`, `CMakeCache.txt`, `(^|/)cache/`, `^[Dd]ependencies/`, `^deps/`},
	},
	`manifests`: &github.FilterInfo{
		Ignore:  false,
		Extract: true,
		Regexp:  []string{`glide\.yaml`, `CMakeLists\.txt`, `package\.json`, `Gopkg\.toml`, `Gomfile`, `req([^.]*)\.txt`, `(^|/)cache/`, `^[Dd]ependencies/`, `^deps/`},
	},
	`docs`: &github.FilterInfo{
		Ignore:  false,
		Extract: true,
		Regexp:  []string{`md`, `markdown`, `mdown`, `mkdn`, `mdwn`, `mdtxt`, `txt`, `text`, `doc`, `htm`, `html`, `pdf`, `ppt`, `pptx`, `docx`},
	},
}

var symbolFiltersMap = map[string][]string{
	`ignore`: {`Godeps`, `vendor`, `node_modules`, `CMakeFiles`, `CMakeScripts`, `CMakeCache.txt`, `(^|/)cache/`, `^[Dd]ependencies/`, `^deps/`},
}
