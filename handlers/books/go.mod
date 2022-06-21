require (
	github.com/aws/aws-lambda-go v1.23.0
	github.com/aws/aws-sdk-go-v2 v1.16.5
	github.com/aws/aws-sdk-go-v2/config v1.15.11
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.9.4
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression v1.4.10
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.15.7
	github.com/google/go-cmp v0.5.8
	github.com/google/gofuzz v1.2.0
)

replace gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.2.8

module books

go 1.16
