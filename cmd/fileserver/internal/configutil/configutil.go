package configutil

type Config struct {
	Hosts   map[string]string `json:"hosts"`
	TLS     TLS               `json:"tls"`
	Servers []Server          `json:"servers"`
}

type TLS struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

type Timeouts struct {
	ReadHeader int `json:"readHeader"`
	Read       int `json:"read"`
	Write      int `json:"write"`
	Idle       int `json:"idle"`
}

type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// FormAuthUser is a single form-auth user; DisplayName is auto-generated if empty.
type FormAuthUser struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

// FormAuth configures form-based session cookie authentication.
type FormAuth struct {
	Users          []FormAuthUser `json:"users"`
	Secret         string         `json:"secret"`
	PublicPrefixes []string       `json:"publicPrefixes"`
}

// ChatChannel is a config-defined channel seeded at startup; users still need the code to join.
type ChatChannel struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// Features controls which optional server features are active.
// All features are enabled by default; set a Disable flag to true to turn one off.
type Features struct {
	ChatRoute     string `json:"chatRoute"` // defaults to "/chat" when empty
	DisableChat   bool   `json:"disableChat"`
	DisableBrowse bool   `json:"disableBrowse"`
}

type Server struct {
	Port             int      `json:"port"`
	AllowCredentials bool     `json:"allowCredentials"`
	MaxAge           int      `json:"maxAge"`
	Timeouts         Timeouts `json:"timeouts"`

	BrowseRateLimit int `json:"browseRateLimit"`
	FileRateLimit   int `json:"fileRateLimit"`

	Hidden []string `json:"hidden"`

	Features  Features  `json:"features"`
	BasicAuth BasicAuth `json:"basicAuth"`
	FormAuth  FormAuth  `json:"formAuth"`

	ChatChannels []ChatChannel `json:"chatChannels"`

	ImageExts        []string `json:"imageExts"`
	TextExts         []string `json:"textExts"`
	ReadmeCandidates []string `json:"readmeCandidates"`

	FileEntries    []FileEntry    `json:"fileEntries"`
	ContentEntries []ContentEntry `json:"contentEntries"`
}

type FileEntry struct {
	Route  string `json:"route"`
	Path   string `json:"path"`
	Browse string `json:"browse"`

	Info Info `json:"info"`

	BasicAuth BasicAuth `json:"basicAuth"`
}

type ContentEntry struct {
	Route  string `json:"route"`
	Name   string `json:"name"`
	Base64 string `json:"base64"` // json marshals []byte as base64; string allows the "asset:" prefix scheme

	Dir     string   `json:"dir"`
	Exclude []string `json:"exclude"`

	Info Info `json:"info"`
}

type Info struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
}
