require (
	github.com/aws/aws-lambda-go v1.23.0
	github.com/aws/aws-sdk-go-v2 v1.16.5
	github.com/aws/aws-sdk-go-v2/config v1.15.10
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.9.3
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.15.6
	github.com/aws/aws-sdk-go-v2/service/ssm v1.27.2
	github.com/google/go-cmp v0.5.8
	github.com/google/gofuzz v1.2.0
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8

module refresh-lists

go 1.16
