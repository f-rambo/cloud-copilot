module github.com/f-rambo/cloud-copilot

go 1.24.3

require (
	github.com/IBM/sarama v1.45.1
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.1.6
	github.com/alibabacloud-go/ecs-20140526/v6 v6.1.0
	github.com/alibabacloud-go/slb-20140515/v4 v4.0.10
	github.com/alibabacloud-go/tea v1.3.6
	github.com/alibabacloud-go/vpc-20160428/v6 v6.12.3
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.12
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.210.1
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.45.0
	github.com/elastic/go-elasticsearch/v9 v9.0.0
	github.com/go-kratos/kratos/contrib/log/zap/v2 v2.0.0-20250429074618-c82f7957223f
	github.com/go-kratos/kratos/v2 v2.8.4
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/google/wire v0.5.0
	github.com/gorilla/handlers v1.5.2
	github.com/joho/godotenv v1.5.1
	github.com/mark3labs/mcp-go v0.30.0
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.13.9
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/common v0.63.0
	github.com/spf13/cast v1.7.1
	go.uber.org/automaxprocs v1.5.3
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.33.0
	golang.org/x/sync v0.11.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240903143218-8af14fe29dc1
	google.golang.org/grpc v1.66.0
	google.golang.org/protobuf v1.36.5
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/postgres v1.5.11
	gorm.io/gorm v1.25.10
	k8s.io/api v0.32.3
	k8s.io/apimachinery v0.32.3
	k8s.io/client-go v0.32.3
)

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/alibabacloud-go/endpoint-util v1.1.0 // indirect
	github.com/alibabacloud-go/openapi-util v0.1.1 // indirect
	github.com/alibabacloud-go/tea-utils/v2 v2.0.7 // indirect
	github.com/aliyun/credentials-go v1.4.5 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.65 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.17 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/eapache/go-resiliency v1.7.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.7.0 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-kratos/aegis v0.2.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-playground/assert/v2 v2.2.0 // indirect
	github.com/go-playground/form/v4 v4.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/oauth2 v0.25.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/term v0.29.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	golang.org/x/time v0.7.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20241105132330-32ad38e42d3f // indirect
	k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738 // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.2 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)
