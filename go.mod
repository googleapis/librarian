module github.com/googleapis/librarian

go 1.25.3

require (
	cloud.google.com/go/artifactregistry v1.17.2
	cloud.google.com/go/cloudbuild v1.23.1
	cloud.google.com/go/iam v1.5.3
	cloud.google.com/go/longrunning v0.7.0
	github.com/cbroglie/mustache v1.4.0
	github.com/go-git/go-git/v5 v5.16.2
	github.com/google/go-cmp v0.7.0
	github.com/google/go-github/v69 v69.2.0
	github.com/google/uuid v1.6.0
	github.com/googleapis/gax-go/v2 v2.15.0
	github.com/iancoleman/strcase v0.3.0
	github.com/pb33f/libopenapi v0.25.9
	github.com/pelletier/go-toml/v2 v2.2.4
	github.com/walle/targz v0.0.0-20140417120357-57fe4206da5a
	github.com/yuin/goldmark v1.7.13
	golang.org/x/exp v0.0.0-20250911091902-df9299821621
	google.golang.org/genproto v0.0.0-20251103181224-f26f9409b101
	google.golang.org/genproto/googleapis/api v0.0.0-20251029180050-ab9386a59fda
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda
	google.golang.org/protobuf v1.36.10
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloud.google.com/go v0.122.0 // indirect
	cloud.google.com/go/auth v0.16.5 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.8.0 // indirect
	dario.cat/mergo v1.0.2 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.3.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.4.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/pb33f/ordered-map/v2 v2.2.0 // indirect
	github.com/pjbgf/sha1cd v0.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/speakeasy-api/jsonpath v0.6.2 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	go.opentelemetry.io/auto/sdk v1.2.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	golang.org/x/crypto v0.42.0 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/oauth2 v0.31.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	golang.org/x/time v0.13.0 // indirect
	google.golang.org/api v0.249.0 // indirect
	google.golang.org/grpc v1.75.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

retract v1.0.0 // We are reverting from a major release as code is not yet stable.
